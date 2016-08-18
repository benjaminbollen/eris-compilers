FROM quay.io/eris/build
MAINTAINER Eris Industries <support@erisindustries.com>

# Install Solc dependencies
RUN apk update && apk add boost-dev build-base cmake jsoncpp-dev

WORKDIR /src 

RUN git clone https://github.com/ethereum/solidity.git --recursive && \
    cd solidity && mkdir build && cd build && cmake .. && \
    make -j 2

ENV INSTALL_BASE /usr/local/bin

# Install Dependencies
RUN apk add ca-certificates curl && \
    update-ca-certificates && \
    rm -rf /var/cache/apk/*
WORKDIR /go

# GO WRAPPER
#ENV GO_WRAPPER_VERSION 1.6
#RUN curl -sSL -o $INSTALL_BASE/go-wrapper https://raw.githubusercontent.com/docker-library/golang/master/$GO_WRAPPER_VERSION/wheezy/go-wrapper
#RUN chmod +x $INSTALL_BASE/go-wrapper

# Install eris-compilers, a go app that manages compilations
ENV REPO $GOPATH/src/github.com/eris-ltd/eris-compilers
#ENV BASE $GOPATH/src/$REPO
#ENV NAME eris-compilers
#RUN mkdir --parents $BASE
COPY . $REPO

WORKDIR $REPO

RUN go get github.com/Masterminds/glide

RUN glide install
RUN cd $REPO/cmd/eris-compilers && go install ./
#RUN unset GOLANG_VERSION && \
#  unset GOLANG_DOWNLOAD_URL && \
#  unset GOLANG_DOWNLOAD_SHA256 && \
#  unset GO_WRAPPER_VERSION && \
#  unset REPO && \
#  unset BASE && \
#  unset NAME && \
#  unset INSTALL_BASE

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
