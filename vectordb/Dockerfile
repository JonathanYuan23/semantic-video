FROM python:3.11-slim

# Install system dependencies required for Pillow and other libraries
RUN apt-get update && apt-get install -y \
    build-essential \
    libjpeg-dev \
    zlib1g-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy requirements and install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY main.py .

# Create directory for Chroma persistence
RUN mkdir -p /data/chroma

# Set environment variable for Chroma path
ENV CHROMA_DB_PATH=/data/chroma

# Expose the port
EXPOSE 8000

# Command to run the service
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]

