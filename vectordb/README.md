# Semantic Video Service

A production-ready service for indexing video frames and searching them via natural language queries using embeddings. Supports both individual image indexing and video frame processing with timestamp-based search results.

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

# Stateless test runs
- Set `STATELESS_MODE=1` (or `STATELESS_TEST=1`) to force Chroma to use a temporary directory that is deleted on shutdown. Example: `docker run -e STATELESS_MODE=1 -p 8000:8000 semantic-video`.
```

The API will be available at `http://localhost:8000`.
Documentation is available at `http://localhost:8000/docs`.

## API Documentation

### Image Endpoints

#### 1. Upload Image
**Endpoint**: `POST /upload_image`

**Description**: Upload and index a single image file for semantic search.

**Parameters**:
- `file` (required): Image file to upload
  - **Type**: File upload (multipart/form-data)
  - **Supported formats**: JPG, PNG, GIF, WebP, etc.
  - **Description**: The image file you want to make searchable

**Request Example**:
```bash
curl -X POST "http://localhost:8000/upload_image" \
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

**Response Fields**:
- `id`: Unique identifier for the uploaded image in the database

---

#### 2. Search Images
**Endpoint**: `POST /search_image`

**Description**: Search for images using natural language queries.

**Parameters**:
- `query` (required): Natural language search query
  - **Type**: String
  - **Example**: "a cat sitting on a chair", "sunset over mountains"
  - **Description**: Describe what you're looking for in plain English
- `top_k` (optional): Number of results to return
  - **Type**: Integer
  - **Default**: 5
  - **Range**: 1-100
  - **Description**: How many matching images to return

**Request Example**:
```bash
curl -X POST "http://localhost:8000/search_image" \
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
      "score": 0.15,
      "metadata": {
        "filename": "cat.jpg",
        "content_type": "image/jpeg",
        "type": "image"
      }
    }
  ]
}
```

**Response Fields**:
- `results`: Array of matching images
  - `id`: Unique identifier of the image
  - `score`: Similarity distance (lower = more similar, 0 = perfect match)
  - `metadata`: Additional information about the image
    - `filename`: Original filename
    - `content_type`: MIME type of the image
    - `type`: Always "image" for image results

---

#### 3. List Images
**Endpoint**: `GET /images`

**Description**: Get a list of all uploaded images.

**Parameters**: None

**Request Example**:
```bash
curl -X GET "http://localhost:8000/images" \
  -H "accept: application/json"
```

**Response**:
```json
{
  "images": ["cat.jpg", "dog.png", "sunset.jpg"]
}
```

**Response Fields**:
- `images`: Array of image filenames that have been uploaded

---

### Video Endpoints

#### 4. Upload Video
**Endpoint**: `POST /upload_video`

**Description**: Upload a video file, automatically extract frames, and index them for search. This is a complete one-step process.

**Parameters**:
- `file` (required): Video file to upload
  - **Type**: File upload (multipart/form-data)
  - **Supported formats**: MP4, MOV, MKV, AVI, M4V, WebM
  - **Description**: The video file you want to make searchable
- `frame_rate` (optional): Frame extraction rate
  - **Type**: Float
  - **Default**: 1.0
  - **Unit**: Frames per second (FPS)
  - **Description**: How many frames to extract per second (1.0 = 1 frame every second, 0.5 = 1 frame every 2 seconds)

**Request Example**:
```bash
curl -X POST "http://localhost:8000/upload_video" \
  -H "accept: application/json" \
  -H "Content-Type: multipart/form-data" \
  -F "file=@/path/to/your/video.mp4" \
  -F "frame_rate=1.0"
```

**Response**:
```json
{
  "video_id": "video_a1b2c3d4",
  "original_filename": "my_video.mp4",
  "saved_as": "video_a1b2c3d4.mp4",
  "status": "success",
  "message": "Video uploaded, frames extracted, and indexed successfully! 120 frames are now searchable.",
  "frames_processed": 120,
  "frames_extracted": 120,
  "ready_for_search": true
}
```

**Response Fields**:
- `video_id`: Unique identifier for the video
- `original_filename`: Name of the uploaded file
- `saved_as`: Name the video was saved as on the server
- `status`: "success" if everything worked
- `message`: Human-readable status message
- `frames_processed`: Number of frames successfully indexed
- `frames_extracted`: Number of frames extracted from video
- `ready_for_search`: Boolean indicating if video is searchable

---

#### 5. Search Videos
**Endpoint**: `POST /search_video`

**Description**: Search for content within videos and get specific timestamp ranges where it appears.

**Parameters**:
- `query` (required): Natural language search query
  - **Type**: String
  - **Example**: "person walking", "car driving", "outdoor scene"
  - **Description**: Describe what you're looking for in the video content
- `top_k` (optional): Number of videos to return
  - **Type**: Integer
  - **Default**: 5
  - **Range**: 1-50
  - **Description**: Maximum number of matching videos to return
- `cluster_threshold` (optional): Timestamp clustering threshold
  - **Type**: Float
  - **Default**: 5.0
  - **Unit**: Seconds
  - **Description**: How close timestamps need to be to group them together (e.g., 5.0 means frames within 5 seconds get grouped into one time range)

**Request Example**:
```bash
curl -X POST "http://localhost:8000/search_video" \
  -H "accept: application/json" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "person walking outdoors",
    "top_k": 5,
    "cluster_threshold": 5.0
  }'
```

**Response**:
```json
{
  "results": [
    {
      "video_id": "video100",
      "video_path": "../videos/video100.mp4",
      "timestamps": [
        {
          "start": 10.0,
          "end": 15.0,
          "relevance_score": 0.85
        },
        {
          "start": 45.0,
          "end": 50.0,
          "relevance_score": 0.78
        }
      ],
      "max_relevance_score": 0.85
    }
  ]
}
```

**Response Fields**:
- `results`: Array of matching videos
  - `video_id`: Unique identifier of the video
  - `video_path`: File path to the video
  - `timestamps`: Array of time ranges where content was found
    - `start`: Start time in seconds
    - `end`: End time in seconds
    - `relevance_score`: How well this time range matches your query (0-1, higher = better match)
  - `max_relevance_score`: Highest relevance score for this video

---

#### 6. List Videos
**Endpoint**: `GET /videos`

**Description**: Get a list of all indexed videos.

**Parameters**: None

**Request Example**:
```bash
curl -X GET "http://localhost:8000/videos" \
  -H "accept: application/json"
```

**Response**:
```json
{
  "videos": ["video100", "video1049", "video_a1b2c3d4"]
}
```

**Response Fields**:
- `videos`: Array of video IDs that have been indexed and are searchable

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
