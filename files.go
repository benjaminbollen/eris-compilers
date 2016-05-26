package compilers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"

	log "github.com/Sirupsen/logrus"
)

// cache compiled regex expressions
var regexCache = make(map[string]*regexp.Regexp)

// Find all matches to the include regex
// Replace filenames with hashes
func (c *CompileClient) replaceIncludes(code []byte, dir string, includes map[string][]byte, includeNames map[string]string) ([]byte, error) {
	// find includes, load those as well
	regexPatterns := c.IncludeRegexes()
	for i, regPattern := range regexPatterns {
		r, ok := regexCache[regPattern]
		if !ok {
			// cache the compiled regex
			var err error
			if r, err = regexp.Compile(regPattern); err != nil {
				return nil, err
			}
			regexCache[regPattern] = r
		}
		// replace all includes with hash of included lll
		//  make sure to return hashes of includes so we can cache check them too
		// do it recursively
		code = r.ReplaceAllFunc(code, func(s []byte) []byte {
			s, err := c.includeReplacer(r, i, s, dir, includes, includeNames)
			if err != nil {
				fmt.Println("ERR!:", err)
				// panic (catch)
			}
			return s
		})
	}

	return code, nil
}

// read the included file, hash it; if we already have it, return include replacement
// if we don't, run replaceIncludes on it (recursive)
// modifies the "includes" map
func (c *CompileClient) includeReplacer(r *regexp.Regexp, i int, s []byte, dir string, included map[string][]byte, includeNames map[string]string) ([]byte, error) {
	m := r.FindSubmatch(s)
	match := m[1]
	// load the file
	p := path.Join(dir, string(match))
	incl_code, err := ioutil.ReadFile(p)
	if err != nil {
		log.Errorln("failed to read include file", err)
		return nil, fmt.Errorf("Failed to read include file: %s", err.Error())
	}

	// take hash before replacing includes to see if we've already parsed this file
	hash := sha256.Sum256(incl_code)
	hpre := hex.EncodeToString(hash[:])
	if h, ok := includeNames[hpre]; ok {
		replaces := c.IncludeReplace(h, i)
		ret := []byte(replaces)
		return ret, nil
	}

	// recursively replace the includes for this file
	this_dir := path.Dir(p)
	incl_code, err = c.replaceIncludes(incl_code, this_dir, included, includeNames)
	if err != nil {
		return nil, err
	}

	// compute hash
	hash = sha256.Sum256(incl_code)
	h := hex.EncodeToString(hash[:])

	replaces := c.IncludeReplace(h, i)
	ret := []byte(replaces)
	included[h] = incl_code
	includeNames[hpre] = h
	return ret, nil
}

// check the cache for all includes, cache those not cached yet
func (c *CompileClient) checkCacheIncludes(includes map[string][]byte) bool {
	cached := true
	for k := range includes {
		f := path.Join(ClientCache, c.Ext(k))
		if _, err := os.Stat(f); err != nil {
			cached = false
			// save empty file named hash of include so we can check
			// whether includes have changed
			ioutil.WriteFile(f, []byte{}, 0644)
		}
	}
	return cached
}

// check/cache all includes, hash the code, return hash and whether or not there was a full cache hit
func (c *CompileClient) checkCached(code []byte, includes map[string][]byte) (string, bool) {
	cachedIncludes := c.checkCacheIncludes(includes)

	// check if the main script has been cached
	hash := sha256.Sum256(code)
	hexHash := hex.EncodeToString(hash[:])
	fname := path.Join(ClientCache, c.Ext(hexHash))
	_, scriptErr := os.Stat(fname)

	// if an include has changed or the script has not been cached
	if !cachedIncludes || scriptErr != nil {
		return hexHash, false
	}
	return hexHash, true
}

// return cached byte code as a response
func (c *CompileClient) cachedResponse(hash string) (*Response, error) {
	d := path.Join(ClientCache, c.Ext(hash))
	var resp *Response
	files, _ := ioutil.ReadDir(d)
	respItemArray := make([]ResponseItem, len(files))
	i := 0
	for _, f := range files {
		fileName := f.Name()
		b, err := ioutil.ReadFile(path.Join(d, fileName, "bin"))
		if err != nil {
			return nil, err
		}
		abi, _ := ioutil.ReadFile(path.Join(d, fileName, "abi"))

		respItem := ResponseItem{
			Objectname: fileName,
			Bytecode:   b,
			ABI:        string(abi),
		}
		respItemArray[i] = respItem
		i++

	}
	resp = &Response{
		Objects: respItemArray,
		Error:   "",
	}

	return resp, nil
}

// cache a file to disk
func (c *CompileClient) cacheFile(b []byte, hash string, name string, ext string) error {
	f := path.Join(ClientCache, c.Ext(hash), name, ext)
	os.MkdirAll(path.Dir(f), 0744)
	if b != nil {
		if err := ioutil.WriteFile(f, b, 0644); err != nil {
			return err
		}
	}
	return nil
}

// check cache for server
func checkCache(objectName string) (*Response, error) {
	f := path.Join(ServerCache, objectName)
	if _, err := os.Stat(f); err == nil {
		b, err := ioutil.ReadFile(f)
		if err != nil {
			return nil, err
		}
		abi, _ := ioutil.ReadFile(f + "-abi")
		return NewResponse(objectName, b, string(abi), nil), nil
	}
	return nil, fmt.Errorf("Not cached")
}

func cacheResult(objectName string, compiled []byte, docs string) {
	f := path.Join(ServerCache, objectName)
	ioutil.WriteFile(f, compiled, 0600)
	ioutil.WriteFile(f+"-abi", []byte(docs), 0600)
}

// Get language from filename extension
func LangFromFile(filename string) (string, error) {
	ext := path.Ext(filename)
	ext = strings.Trim(ext, ".")
	if _, ok := Languages[ext]; ok {
		return ext, nil
	}
	for l, lc := range Languages {
		for _, e := range lc.Extensions {
			if ext == e {
				return l, nil
			}
		}
	}
	return "", UnknownLang(ext)
}

// the string is not literal if it ends in a valid extension
func isLiteral(f, lang string) bool {
	if strings.HasSuffix(f, Languages[lang].Ext("")) {
		return false
	}

	for _, lc := range Languages {
		for _, e := range lc.Extensions {
			if strings.HasSuffix(f, e) {
				return false
			}
		}
	}
	return true
}

// Clear client and server caches
func ClearCaches() error {
	if err := ClearServerCache(); err != nil {
		return err
	}
	return ClearClientCache()
}

// clear a directory of its contents
func clearDir(dir string) error {
	fs, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, f := range fs {
		n := f.Name()
		if err := os.Remove(path.Join(dir, n)); err != nil {
			return err
		}
	}
	return nil
}

// Clear the server cache
func ClearServerCache() error {
	return clearDir(ServerCache)
}

// Clear the client cache
func ClearClientCache() error {
	return clearDir(ClientCache)
}
