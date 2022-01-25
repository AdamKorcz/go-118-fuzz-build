FROM gcr.io/oss-fuzz-base/base-builder-go
RUN git clone --depth 1 --branch fuzz32 https://github.com/AdamKorcz/vitess
COPY build.sh $SRC/
WORKDIR $SRC/vitess
