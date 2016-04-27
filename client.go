package compilers

import (
	"io/ioutil"
	"path"

	log "github.com/eris-ltd/eris-compilers/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/eris-ltd/eris-compilers/Godeps/_workspace/src/github.com/eris-ltd/common/go/common"
)

// ClientCache location in eris tree
var ClientCache = path.Join(common.LllcScratchPath, "client")

// filename is either a filename or literal code
func resolveCode(filename string, literal bool) (code []byte, err error) {
	if !literal {
		code, err = ioutil.ReadFile(filename)
	} else {
		code = []byte(filename)
	}
	log.Debugf("Code that is read =>\t%s\n", code)
	return
}

// send compile request to server or compile directly
func (c *CompileClient) compileRequest(req *Request) (resp *Response, err error) {
	log.Info("Made it to compile request")
	if c.config.Net {
		log.WithField("url", c.config.URL).Debug("Compiling code remotely")
		resp, err = requestResponse(req)
	} else {
		log.Debug("Compiling code locally.")
		resp = compileServerCore(req)
	}
	return
}

// Compile takes a dir and some code, replaces all includes, checks cache, compiles, caches
func (c *CompileClient) Compile(dir string, code []byte, libraries string) (*Response, error) {
	// replace includes with hash of included contents and add those contents to Includes (recursive)
	var includes = make(map[string][]byte)     // hashes to code
	var includeNames = make(map[string]string) //hashes before replace to hashes after
	var err error
	// log.Debugf("Before parsing includes =>\n\n%s", string(code))
	code, err = c.replaceIncludes(code, dir, includes, includeNames)
	if err != nil {
		return nil, err
	}
	// log.Debugf("After parsing includes =>\t\t%s\n\n%s", includes, string(code))

	// go through all includes, check if they have changed
	hash, cached := c.checkCached(code, includes)

	log.WithFields(log.Fields{
		"hash":    hash,
		"cached?": cached,
	}).Debug("File to compile")

	// if everything is cached, no need for request
	if cached {
		// TODO: need to return all contracts/libs tied to the original src file
		return c.cachedResponse(hash)
	}
	req := NewRequest(code, includes, c.Lang(), libraries)

	// response struct (returned)
	resp, err := c.compileRequest(req)

	if err != nil {
		log.Info("Error in compile request")
		return nil, err
	}
	log.Info("RESPONSE ERROR: ", resp.Error)
	if resp.Error == "" {
		for _, r := range resp.Objects {
			// fill in cached values, cache new values
			if r.Bytecode != nil {
				if err := c.cacheFile(r.Bytecode, hash, r.Objectname, "bin"); err != nil {
					return nil, err
				}
				if err := c.cacheFile([]byte(r.ABI), hash, r.Objectname, "abi"); err != nil {
					return nil, err
				}
			}
		}
	}

	return resp, nil
}

// create a new compiler for the language and compile the code
func compile(filename string, code []byte, lang, dir string, libraries string) *Response {
	c, err := NewCompileClient(lang)
	if err != nil {
		return NewResponse("", nil, "", err)
	}
	r, err := c.Compile(dir, code, libraries)
	if err != nil {
		return NewResponse("", nil, "", err)
	}

	return r
}

// Compile a file and resolve includes
func Compile(filename string, libraries string) *Response {
	lang, err := LangFromFile(filename)
	if err != nil {
		return NewResponse("", nil, "", err)
	}

	log.WithField("=>", lang).Info("Language to use")

	code, err := ioutil.ReadFile(filename)
	if err != nil {
		return NewResponse("", nil, "", err)

	}
	dir := path.Dir(filename)
	return compile(filename, code, lang, dir, libraries)
}

// Compile a literal piece of code
func CompileLiteral(code string, lang string) *Response {
	return compile("Literal", []byte(code), lang, common.LllcScratchPath, "")
}
