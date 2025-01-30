FROM ubuntu:24.10

ARG DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y python3 python3-pip
RUN apt-get install -y python3-venv

#    python3-pip \
#    && rm -rf /var/lib/apt/lists/*

WORKDIR /sandboxai
RUN python3 -m venv venv
ENV PATH="/sandboxai/venv/bin:$PATH"

COPY ./python/boxd/requirements.txt ./
RUN pip install -r requirements.txt

COPY ./python/boxd ./boxd
COPY ./python/sandboxai ./sandboxai

WORKDIR /work

CMD ["uvicorn", "boxd.main:app", "--app-dir=/sandboxai", "--host=0.0.0.0"]
