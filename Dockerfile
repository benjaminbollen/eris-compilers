FROM quay.io/eris/base:alpine
MAINTAINER Eris Industries <support@erisindustries.com>

# Install Solc dependencies
RUN apk --no-cache --update add --virtual dependencies \
            libgcc \
            libstdc++
RUN apk --no-cache --update add --virtual build-dependencies \
            bash \
            cmake \
            curl-dev \
            git \
            gcc \
            g++ \
            linux-headers \
            make \
            perl \
            python \
            scons\

            boost-dev \
            gmp-dev\
            libmicrohttpd-dev

RUN mkdir -p /src/deps

WORKDIR /src/deps

RUN git clone https://github.com/mmoss/cryptopp.git
RUN git clone https://github.com/open-source-parsers/jsoncpp.git
RUN git clone https://github.com/cinemast/libjson-rpc-cpp
RUN git clone https://github.com/google/leveldb

ENV PREFIX /src/built

RUN mkdir -p ${PREFIX}/include ${PREFIX}/lib

RUN cd cryptopp \
 && cmake -DCRYPTOPP_LIBRARY_TYPE=STATIC \
          -DCRYPTOPP_RUNTIME_TYPE=STATIC \
          -DCRYPTOPP_BUILD_TESTS=FALSE \
          -DCMAKE_INSTALL_PREFIX=${PREFIX}/ \
          . \
 && make cryptlib \
 && cp -r src ${PREFIX}/include/cryptopp \
 && cp src/libcryptlib.a ${PREFIX}/lib/


## These aren't really necessary for solc, but can't build without them
## as devcore links to them.
RUN cd jsoncpp \
 && cmake -DCMAKE_INSTALL_PREFIX=${PREFIX}/ . \
 && make jsoncpp_lib_static \
 && make install

RUN mkdir -p libjson-rpc-cpp/build \
 && sed -e 's/^#include <string>/#include <string.h>/' libjson-rpc-cpp/src/jsonrpccpp/server/connectors/unixdomainsocketserver.cpp -i \
 && cd libjson-rpc-cpp/build \
 && cmake -DJSONCPP_LIBRARY=../../jsoncpp/src/lib_json/libjsoncpp.a \
          -DJSONCPP_INCLUDE_DIR=../../jsoncpp/include/ \
          -DBUILD_STATIC_LIBS=YES                      \
          -DBUILD_SHARED_LIBS=NO                       \
          -DCOMPILE_TESTS=NO                           \
          -DCOMPILE_EXAMPLES=NO                        \
          -DCOMPILE_STUBGEN=NO                         \
          -DCMAKE_INSTALL_PREFIX=${PREFIX}/           \
          .. \
 && make install

RUN cd leveldb \
 && make \
 && cp -rv include/leveldb ${PREFIX}/include/ \
 && cp -v out-static/libleveldb.a ${PREFIX}/lib/

WORKDIR /src 

RUN git clone https://github.com/ethereum/solidity.git && \
    cd solidity && mkdir build && cd build && cmake .. && \
    make -j 4

ENV INSTALL_BASE /usr/local/bin

# Install Dependencies
RUN apk add ca-certificates curl && \
    update-ca-certificates && \
    rm -rf /var/cache/apk/*
WORKDIR /go

# GO WRAPPER
ENV GO_WRAPPER_VERSION 1.6
RUN curl -sSL -o $INSTALL_BASE/go-wrapper https://raw.githubusercontent.com/docker-library/golang/master/$GO_WRAPPER_VERSION/wheezy/go-wrapper
RUN chmod +x $INSTALL_BASE/go-wrapper

# GLIDE INSTALL
RUN add-apt-repository ppa:masterminds/glide \
  && apt-get update
RUN apt-get install glide

# Install eris-compilers, a go app that manages compilations
ENV REPO github.com/eris-ltd/eris-compilers
ENV BASE $GOPATH/src/$REPO
ENV NAME eris-compilers
RUN mkdir --parents $BASE
COPY . $BASE/
RUN cd $BASE && glide install && \
  cd $BASE/cmd/$NAME && go install ./
RUN unset GOLANG_VERSION && \
  unset GOLANG_DOWNLOAD_URL && \
  unset GOLANG_DOWNLOAD_SHA256 && \
  unset GO_WRAPPER_VERSION && \
  unset REPO && \
  unset BASE && \
  unset NAME && \
  unset INSTALL_BASE

# Setup User
ENV USER eris
ENV ERISDIR /home/$USER/.eris

# Add Gandi certs for eris
COPY docker/gandi2.crt /data/gandi2.crt
COPY docker/gandi3.crt /data/gandi3.crt
RUN chown --recursive $USER /data

# Copy in start script
COPY docker/start.sh /home/$USER/

# Point to the compiler location.
RUN chown --recursive $USER:$USER /home/$USER

# Finalize
USER $USER
VOLUME $ERISDIR
WORKDIR /home/$USER
EXPOSE 9098 9099
CMD ["/home/eris/start.sh"]
