# Semantic Video Service (currently only for images)

A minimal production-ready service for indexing images and searching them via natural language queries using embeddings.

## Technology Stack

- **Language**: Python 3.11+
- **Framework**: FastAPI
- **Embeddings**: `sentence-transformers` (clip-ViT-B-32)
- **Vector DB**: Chroma (local persistent)
- **Container**: Docker

## Building the Docker Image

Run the following command in the root directory of the project:

```bash
docker build -t semantic-video .
```

## Running the Container

To run the service with data persistence (so your vector index is saved), mount a local directory to `/data/chroma` inside the container.

```bash
# Create a local directory for data if it doesn't exist
mkdir -p chroma-data

# Run the container (Linux/macOS/PowerShell)
docker run -p 8000:8000 -v "${PWD}/chroma-data:/data/chroma" semantic-video

# Run the container (Windows Command Prompt)
# docker run -p 8000:8000 -v "%cd%/chroma-data:/data/chroma" semantic-video
```

The API will be available at `http://localhost:8000`.
Documentation is available at `http://localhost:8000/docs`.

## Usage

### 1. Index an Image

Upload an image file to be embedded and stored.

**Endpoint**: `POST /index_image`

**Curl Example**:

```bash
curl -X POST "http://localhost:8000/index_image" \
  -H "accept: application/json" \
  -H "Content-Type: multipart/form-data" \
  -F "file=@/path/to/your/image.jpg"
```

**Response**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### 2. Batch Index Images

Upload multiple image files at once.

**Endpoint**: `POST /index_batch`

**Curl Example**:

```bash
curl -X POST "http://localhost:8000/index_batch" \
  -H "accept: application/json" \
  -H "Content-Type: multipart/form-data" \
  -F "files=@image1.jpg" \
  -F "files=@image2.jpg"
```

**Response**:
```json
{
  "results": [
    {
      "filename": "image1.jpg",
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "status": "success",
      "error": null
    },
    {
      "filename": "image2.jpg",
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "status": "success",
      "error": null
    }
  ]
}

### 3. Search Images

Search for images using a text query.

**Endpoint**: `POST /search`

**Curl Example**:

```bash
curl -X POST "http://localhost:8000/search" \
  -H "accept: application/json" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "a photo of a cat",
    "top_k": 3
  }'
```

**Response**:
```json
{
  "results": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "score": 0.85,
      "metadata": {
        "filename": "cat.jpg",
        "content_type": "image/jpeg"
      }
    }
  ]
}
```

## Development

To run locally without Docker:

1. Install dependencies:
   ```bash
   pip install -r requirements.txt
   ```

2. Run the server:
   ```bash
   uvicorn main:app --reload
   ```

