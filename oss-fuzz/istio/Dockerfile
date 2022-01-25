FROM gcr.io/oss-fuzz-base/base-builder-go
RUN git clone --depth 1 --branch fuzz1 https://github.com/AdamKorcz/istio
COPY build.sh $SRC/
WORKDIR $SRC/istio
