
FROM gcr.io/oss-fuzz-base/base-builder-go

RUN git clone --single-branch --depth=1 https://github.com/ethereum/go-ethereum $GOPATH/src/github.com/ethereum/go-ethereum 
RUN (cd $GOPATH/src/github.com/ethereum/go-ethereum && go mod download)

RUN cp $GOPATH/src/github.com/ethereum/go-ethereum/oss-fuzz.sh $SRC/build.sh
# Enable this for easier local testing / repro
#ADD build.sh $SRC/build.sh

WORKDIR $SRC/

