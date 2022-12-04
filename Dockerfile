FROM golang:1.18
# Add a work directory
WORKDIR /app

RUN apt update --fix-missing && apt install -y
RUN apt install python -y
RUN apt install ffmpeg -y
RUN apt install mediainfo -y
RUN apt install unzip -y
ADD https://github.com/kynesim/tstools/archive/refs/heads/master.zip ./
RUN unzip master.zip
RUN cd  tstools-master/ && make && make install

# Cache and install dependencies
COPY go.mod go.sum ./

RUN go get