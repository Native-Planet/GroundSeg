FROM python:3.10.7-slim-buster

RUN echo "deb http://deb.debian.org/debian buster-backports main non-free" >> /etc/apt/sources.list
RUN apt-get update
RUN apt-get install -y curl python3-pip wireguard ca-certificates gnupg lsb-release network-manager
RUN service network-manager start

RUN mkdir -p /etc/apt/keyrings
RUN curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
RUN echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian $(lsb_release -cs) stable" 
RUN echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
RUN apt-get update
RUN apt-get -y install docker-ce docker-ce-cli containerd.io docker-compose-plugin
COPY ./ /app
RUN curl https://dl.min.io/client/mc/release/linux-amd64/mc -o /app/mc
RUN pip3 install -r /app/requirements.txt
RUN mkdir -p /settings/pier
RUN mkdir -p /app/tmp

EXPOSE 27016
ENTRYPOINT ["python3","/app/app.py"]
