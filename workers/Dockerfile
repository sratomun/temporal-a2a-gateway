FROM python:3.12-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements
COPY workers/requirements.txt .

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy SDK (for transition phase)
COPY python-sdk /app/python-sdk

# Copy worker code
COPY workers/echo_worker.py .
COPY workers/echo_logic.py .

# Run the worker
CMD ["python", "echo_worker.py"]