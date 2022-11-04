FROM python:3.10.7-slim-buster

RUN echo "deb http://deb.debian.org/debian buster-backports main non-free" >> /etc/apt/sources.list
RUN apt-get update
RUN apt-get install -y curl python3-pip wireguard ca-certificates network-manager
RUN service network-manager start

COPY ./ /app

RUN curl https://dl.min.io/client/mc/release/linux-amd64/mc -o /app/mc
RUN pip3 install -r /app/requirements.txt

RUN mkdir -p /settings/pier
RUN mkdir -p /app/tmp

EXPOSE 27016
ENTRYPOINT ["python3","/app/app.py"]
