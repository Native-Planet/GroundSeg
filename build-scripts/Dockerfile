from ubuntu:20.04

RUN apt-get update
RUN apt-get install -y make build-essential libssl-dev zlib1g-dev \
       libbz2-dev libreadline-dev libsqlite3-dev wget curl llvm \
       libncurses5-dev libncursesw5-dev xz-utils tk-dev wget \
       libtk8.6 libgdm-dev libdb4o-cil-dev libpcap-dev \
       patchelf clang ca-certificates

RUN update-ca-certificates --fresh
RUN export SSL_CERT_DIR=/etc/ssl/certs

RUN wget https://www.python.org/ftp/python/3.10.9/Python-3.10.9.tgz
RUN tar xzvf Python-3.10.9.tgz

WORKDIR /Python-3.10.9

RUN ./configure --enable-optimizations --with-ensurepip=install
RUN make -j 8
RUN make altinstall

WORKDIR /
# Copy the build script
COPY ./builder.sh /builder.sh
RUN chmod +x /builder.sh

CMD ./builder.sh
