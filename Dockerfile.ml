# ML Service Dockerfile
FROM python:3.11-slim

# Set working directory
WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements first for better caching
COPY ml-service/requirements.txt .

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Install grpcio-tools for proto generation
RUN pip install --no-cache-dir grpcio-tools==1.62.0

# Copy ML service code
COPY ml-service/ .
COPY proto/ /app/proto/

# Generate gRPC code
RUN mkdir -p /app/generated && \
    touch /app/generated/__init__.py && \
    python -m grpc_tools.protoc \
    -I/app/proto/ml \
    --python_out=/app/generated \
    --grpc_python_out=/app/generated \
    ml_service.proto

# Create models directory structure
RUN mkdir -p /app/models/saved

# Create non-root user
RUN useradd -m -u 1000 mluser && \
    chown -R mluser:mluser /app

USER mluser

# Expose gRPC port (using 50055 as an uncommon port)
EXPOSE 50055

# Run the ML service
CMD ["python", "main.py", "--port", "50055"]
