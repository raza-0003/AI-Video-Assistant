FROM python:3.11-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    build-essential \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Longer timeout + retries: default 15s times out on slow/flaky connections
# mid-download of large ML packages, which is what was failing before.
ENV PIP_DEFAULT_TIMEOUT=200
ENV PIP_RETRIES=10

RUN pip install --no-cache-dir --upgrade pip

# CPU-only PyTorch build — the default PyPI wheel bundles CUDA and is
# 5-10x larger for no benefit in a container with no GPU. This alone fixes
# most download-timeout failures.
RUN pip install --no-cache-dir torch torchaudio --index-url https://download.pytorch.org/whl/cpu

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt fastapi "uvicorn[standard]"

COPY . .

EXPOSE 8000
CMD ["uvicorn", "service:app", "--host", "0.0.0.0", "--port", "8000"]
