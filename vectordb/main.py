import os
import io
import uuid
import re
import shutil
import tempfile
from typing import List, Optional, Dict
import logging
from pathlib import Path
from collections import defaultdict
import cv2
import numpy as np

from fastapi import FastAPI, UploadFile, File, HTTPException, Form
from pydantic import BaseModel
from PIL import Image
from sentence_transformers import SentenceTransformer
import chromadb
from chromadb.config import Settings

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="Semantic Video API")

# Global variables for model and vector db
model: SentenceTransformer = None
chroma_client: chromadb.PersistentClient = None
collection: chromadb.Collection = None

MODEL_NAME = "clip-ViT-B-32"
STATELESS_MODE = os.getenv("STATELESS_TEST") == "1" or os.getenv("STATELESS_MODE") == "1"
DEFAULT_CHROMA_PATH = "chroma_data"
# Use environment variable for path, default to local relative path
CHROMA_DB_PATH = os.getenv("CHROMA_DB_PATH", DEFAULT_CHROMA_PATH)
_CLEANUP_CHROMA_PATH: Optional[str] = None
if STATELESS_MODE:
    CHROMA_DB_PATH = tempfile.mkdtemp(prefix="chroma-")
    _CLEANUP_CHROMA_PATH = CHROMA_DB_PATH
COLLECTION_NAME = "images"

# Pydantic Models
class IndexImageResponse(BaseModel):
    id: str

class SearchRequest(BaseModel):
    query: str
    top_k: int = 5

class SearchResultItem(BaseModel):
    id: str
    score: Optional[float] = None
    metadata: Optional[dict] = None

class SearchResponse(BaseModel):
    results: List[SearchResultItem]

class VideoSearchRequest(BaseModel):
    query: str
    top_k: int = 5
    cluster_threshold: float = 5.0  # kept for compatibility, unused now

class TimestampRange(BaseModel):
    start: float
    end: float
    relevance_score: float

class VideoSearchResult(BaseModel):
    video_id: str
    video_path: str
    timestamps: List[TimestampRange]
    max_relevance_score: float

class VideoSearchResponse(BaseModel):
    results: List[VideoSearchResult]

class VideoListResponse(BaseModel):
    videos: List[str]

class ImageListResponse(BaseModel):
    images: List[str]

@app.on_event("startup")
async def startup_event():
    """
    Load the model and initialize the vector database connection on startup.
    """
    global model, chroma_client, collection
    
    logger.info(f"Loading model: {MODEL_NAME}...")
    try:
        model = SentenceTransformer(MODEL_NAME)
    except Exception as e:
        logger.error(f"Failed to load model: {e}")
        raise RuntimeError("Could not load embedding model.")

    logger.info(f"Initializing ChromaDB at {CHROMA_DB_PATH}...")
    try:
        chroma_client = chromadb.PersistentClient(path=CHROMA_DB_PATH)
        collection = chroma_client.get_or_create_collection(
            name=COLLECTION_NAME,
            metadata={"hnsw:space": "cosine"}
        )
    except Exception as e:
        logger.error(f"Failed to initialize ChromaDB: {e}")
        raise RuntimeError("Could not initialize vector database.")
        
    logger.info("Service startup complete.")


@app.on_event("shutdown")
async def shutdown_event():
    """Cleanup temporary Chroma storage when running in stateless mode."""
    global _CLEANUP_CHROMA_PATH
    if _CLEANUP_CHROMA_PATH and os.path.isdir(_CLEANUP_CHROMA_PATH):
        try:
            shutil.rmtree(_CLEANUP_CHROMA_PATH, ignore_errors=True)
            logger.info(f"Removed temporary Chroma data at {_CLEANUP_CHROMA_PATH}")
        except Exception as e:
            logger.error(f"Failed to remove Chroma data at shutdown: {e}")

# Helper Functions
def embed_image(image_bytes: bytes) -> List[float]:
    """
    Load bytes as RGB image and generate embedding using the CLIP model.
    """
    try:
        image = Image.open(io.BytesIO(image_bytes)).convert("RGB")
        # normalize_embeddings=True is common for cosine similarity searches
        embedding = model.encode(image, normalize_embeddings=True)
        return embedding.tolist()
    except Exception as e:
        logger.error(f"Error embedding image: {e}")
        raise ValueError("Invalid image data")

def embed_text(text: str) -> List[float]:
    """
    Generate text embedding using the CLIP model.
    """
    try:
        embedding = model.encode(text, normalize_embeddings=True)
        return embedding.tolist()
    except Exception as e:
        logger.error(f"Error embedding text: {e}")
        raise ValueError("Error generating text embedding")

def extract_frame_info(frame_filename: str) -> tuple[int, str]:
    """
    Extract frame number and video ID from frame filename.
    Expected format: frame_00001.jpg, frame_00002.jpg, etc.
    """
    try:
        # Extract frame number from filename like "frame_00001.jpg"
        match = re.match(r'frame_(\d+)\.jpg', frame_filename)
        if not match:
            raise ValueError(f"Invalid frame filename format: {frame_filename}")
        
        frame_number = int(match.group(1))
        return frame_number, frame_filename
    except Exception as e:
        logger.error(f"Error extracting frame info from {frame_filename}: {e}")
        raise ValueError(f"Invalid frame filename: {frame_filename}")

def calculate_timestamp(frame_number: int, frame_rate: float) -> float:
    """
    Calculate timestamp from frame number and frame rate.
    Frame numbering starts at 1, so frame 1 = timestamp 0.
    """
    return (frame_number - 1) / frame_rate

def cluster_timestamps(timestamps: List[float], threshold: float = 5.0) -> List[tuple[float, float]]:
    """
    Group nearby timestamps into ranges.
    Returns list of (start, end) timestamp ranges.
    """
    if not timestamps:
        return []
    
    timestamps = sorted(timestamps)
    ranges = []
    current_start = timestamps[0]
    current_end = timestamps[0]
    
    for ts in timestamps[1:]:
        if ts - current_end <= threshold:
            current_end = ts
        else:
            ranges.append((current_start, current_end))
            current_start = ts
            current_end = ts
    
    ranges.append((current_start, current_end))
    return ranges

def extract_frames_from_video(video_path: str, output_dir: str, frame_rate: float = 1.0) -> List[str]:
    """
    Extract frames from video using OpenCV and save them as JPG files.
    Returns list of frame filenames.
    """
    try:
        # Create output directory
        output_path = Path(output_dir)
        output_path.mkdir(parents=True, exist_ok=True)
        
        # Open video
        cap = cv2.VideoCapture(video_path)
        if not cap.isOpened():
            raise ValueError(f"Could not open video file: {video_path}")
        
        # Get video properties
        fps = cap.get(cv2.CAP_PROP_FPS)
        total_frames = int(cap.get(cv2.CAP_PROP_FRAME_COUNT))
        
        logger.info(f"Video FPS: {fps}, Total frames: {total_frames}")
        
        # Calculate frame interval based on desired frame rate
        frame_interval = int(fps / frame_rate) if frame_rate < fps else 1
        
        frame_files = []
        frame_count = 0
        saved_count = 0
        
        while True:
            ret, frame = cap.read()
            if not ret:
                break
            
            # Save frame at specified intervals
            if frame_count % frame_interval == 0:
                saved_count += 1
                frame_filename = f"frame_{saved_count:05d}.jpg"
                frame_path = output_path / frame_filename
                
                # Save frame as JPG
                cv2.imwrite(str(frame_path), frame)
                frame_files.append(frame_filename)
                
                logger.info(f"Extracted frame {saved_count}: {frame_filename}")
            
            frame_count += 1
        
        cap.release()
        logger.info(f"Extracted {len(frame_files)} frames from video")
        
        return sorted(frame_files)
        
    except Exception as e:
        logger.error(f"Error extracting frames: {e}")
        raise ValueError(f"Failed to extract frames: {str(e)}")

def get_video_frames(frames_directory: str, video_id: str) -> List[str]:
    """
    Get all frame files for a specific video from the frames directory.
    """
    video_frames_dir = Path(frames_directory) / video_id
    if not video_frames_dir.exists():
        raise ValueError(f"Video frames directory not found: {video_frames_dir}")
    
    frame_files = []
    for frame_file in video_frames_dir.glob("frame_*.jpg"):
        frame_files.append(frame_file.name)
    
    return sorted(frame_files)

# Endpoints
@app.post("/upload_image", response_model=IndexImageResponse)
async def upload_image(
    file: UploadFile = File(...),
    video_id: Optional[str] = Form(None),
    video_path: Optional[str] = Form(None),
    timestamp: Optional[float] = Form(None),
    frame_number: Optional[int] = Form(None),
    frame_rate: Optional[float] = Form(None),
):
    """
    Upload an image file and index it into the database for search.
    """
    if not file.filename:
        raise HTTPException(status_code=400, detail="No file provided.")

    try:
        contents = await file.read()
        if not contents:
            raise HTTPException(status_code=400, detail="Empty file.")
            
        # Generate embedding
        try:
            embedding = embed_image(contents)
        except ValueError:
             raise HTTPException(status_code=400, detail="Invalid image format or corrupted file.")

        # Generate ID and metadata
        image_id = str(uuid.uuid4())
        is_video_frame = bool(video_id)
        metadata = {
            "filename": file.filename,
            "content_type": file.content_type or "unknown",
            "type": "video_frame" if is_video_frame else "image",
        }

        if is_video_frame:
            metadata.update(
                {
                    "video_id": video_id,
                    "video_path": video_path or "",
                    "frame_number": frame_number,
                    "timestamp": timestamp,
                    "frame_rate": frame_rate,
                }
            )

        # Store in Chroma
        collection.add(
            ids=[image_id],
            embeddings=[embedding],
            metadatas=[metadata]
        )
        
        logger.info(f"Successfully indexed image: {file.filename}")
        return IndexImageResponse(id=image_id)

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Unexpected error in /upload_image: {e}")
        raise HTTPException(status_code=500, detail="Internal server error processing image.")

@app.post("/search_image", response_model=SearchResponse)
async def search_image(request: SearchRequest):
    """
    Search for images using natural language queries.
    """
    if not request.query.strip():
        raise HTTPException(status_code=400, detail="Query cannot be empty.")

    try:
        # Embed query
        query_embedding = embed_text(request.query)

        # Search Chroma for images only
        results = collection.query(
            query_embeddings=[query_embedding],
            n_results=request.top_k,
            include=["metadatas", "distances"],
            where={"type": "image"}  # Only search regular images
        )

        # Format response
        ids = results["ids"][0]
        distances = results["distances"][0] if results["distances"] else []
        metadatas = results["metadatas"][0] if results["metadatas"] else []

        response_items = []
        for i in range(len(ids)):
            item = SearchResultItem(
                id=ids[i],
                score=distances[i] if i < len(distances) else None,
                metadata=metadatas[i] if i < len(metadatas) else None
            )
            response_items.append(item)

        return SearchResponse(results=response_items)

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Unexpected error in /search_image: {e}")
        raise HTTPException(status_code=500, detail="Internal server error processing search.")

@app.get("/images", response_model=ImageListResponse)
async def list_images():
    """
    List all indexed images.
    """
    try:
        # Query all images to get unique image IDs
        results = collection.query(
            query_embeddings=[embed_text("image")],  # Dummy query
            n_results=10000,  # Large number to get all results
            include=["metadatas"],
            where={"type": "image"}
        )
        
        image_ids = []
        metadatas = results["metadatas"][0] if results["metadatas"] else []
        
        for i, metadata in enumerate(metadatas):
            if metadata and "filename" in metadata:
                image_ids.append(metadata["filename"])
        
        return ImageListResponse(images=sorted(list(set(image_ids))))
        
    except Exception as e:
        logger.error(f"Error listing images: {e}")
        raise HTTPException(status_code=500, detail="Internal server error listing images.")

@app.post("/upload_video")
async def upload_video(file: UploadFile = File(...), frame_rate: float = 1.0):
    """
    Upload a video file and automatically index all frames into the database.
    This handles: upload → save → extract frames → index frames → ready for search
    """
    if not file.filename:
        raise HTTPException(status_code=400, detail="No file provided.")
    
    # Validate file type
    allowed_extensions = {".mp4", ".mov", ".mkv", ".avi", ".m4v", ".webm"}
    file_ext = Path(file.filename).suffix.lower()
    
    if file_ext not in allowed_extensions:
        raise HTTPException(status_code=400, detail=f"Unsupported video format: {file_ext}")
    
    # Generate video ID
    video_id = f"video_{uuid.uuid4().hex[:8]}"
    
    try:
        # Step 1: Save video file
        videos_dir = Path("../videos")
        frames_dir = Path("../frames")
        
        videos_dir.mkdir(exist_ok=True)
        frames_dir.mkdir(exist_ok=True)
        
        video_filename = f"{video_id}{file_ext}"
        video_path = videos_dir / video_filename
        
        logger.info(f"Saving video {video_id} to {video_path}")
        
        contents = await file.read()
        with open(video_path, "wb") as f:
            f.write(contents)
        
        # Step 2: Extract frames directly from the uploaded video
        video_frames_dir = frames_dir / video_id
        
        logger.info(f"Extracting frames from video {video_id}...")
        
        # Extract frames using OpenCV
        frame_files = extract_frames_from_video(
            str(video_path), 
            str(video_frames_dir), 
            frame_rate
        )
        
        if not frame_files:
            raise HTTPException(status_code=500, detail="Failed to extract any frames from video")
        
        # Step 3: Index all extracted frames
        logger.info(f"Indexing {len(frame_files)} frames for {video_id}...")
        
        # Prepare lists for batch adding to Chroma
        ids_to_add = []
        embeddings_to_add = []
        metadatas_to_add = []
        
        frames_processed = 0
        
        for frame_file in frame_files:
            try:
                # Extract frame info
                frame_number, _ = extract_frame_info(frame_file)
                timestamp = calculate_timestamp(frame_number, frame_rate)
                
                # Read and embed frame
                frame_path = video_frames_dir / frame_file
                with open(frame_path, 'rb') as f:
                    frame_bytes = f.read()
                
                embedding = embed_image(frame_bytes)
                
                # Create metadata with video information
                frame_id = str(uuid.uuid4())
                metadata = {
                    "filename": frame_file,
                    "content_type": "image/jpeg",
                    "video_id": video_id,
                    "video_path": str(video_path),
                    "frame_number": frame_number,
                    "timestamp": timestamp,
                    "frame_rate": frame_rate,
                    "type": "video_frame"
                }
                
                ids_to_add.append(frame_id)
                embeddings_to_add.append(embedding)
                metadatas_to_add.append(metadata)
                frames_processed += 1
                
            except Exception as e:
                logger.error(f"Error processing frame {frame_file}: {e}")
                continue
        
        if not ids_to_add:
            raise HTTPException(status_code=500, detail="Failed to process any frames")
        
        # Batch add to Chroma
        collection.add(
            ids=ids_to_add,
            embeddings=embeddings_to_add,
            metadatas=metadatas_to_add
        )
        
        logger.info(f"Successfully indexed {frames_processed} frames for video {video_id}")
        
        return {
            "video_id": video_id,
            "original_filename": file.filename,
            "saved_as": video_filename,
            "status": "success",
            "message": f"Video uploaded, frames extracted, and indexed successfully! {frames_processed} frames are now searchable.",
            "frames_processed": frames_processed,
            "frames_extracted": len(frame_files),
            "ready_for_search": True
        }
        
    except Exception as e:
        logger.error(f"Error uploading video: {e}")
        raise HTTPException(status_code=500, detail=f"Error uploading video: {str(e)}")

@app.post("/search_video", response_model=VideoSearchResponse)
async def search_video(request: VideoSearchRequest):
    """
    Search for videos containing the query and return results grouped by video with timestamps.
    """
    if not request.query.strip():
        raise HTTPException(status_code=400, detail="Query cannot be empty.")
    
    try:
        # Embed query
        query_embedding = embed_text(request.query)
        
        # Search Chroma for video frames only
        results = collection.query(
            query_embeddings=[query_embedding],
            n_results=request.top_k * 10,  # Get more results to group by video
            include=["metadatas", "distances"],
            where={"type": "video_frame"}  # Only search video frames
        )
        
        # Group results by video
        video_results = defaultdict(list)
        
        ids = results["ids"][0]
        distances = results["distances"][0] if results["distances"] else []
        metadatas = results["metadatas"][0] if results["metadatas"] else []
        
        for i in range(len(ids)):
            if i < len(metadatas) and metadatas[i]:
                metadata = metadatas[i]
                video_id = metadata.get("video_id")
                timestamp = metadata.get("timestamp", 0)
                distance = distances[i] if i < len(distances) else 0
                # Convert cosine distance (0 best) to similarity in [0,1]
                score = max(0.0, 1.0 - distance)
                
                if video_id:
                    video_results[video_id].append({
                        "timestamp": timestamp,
                        "score": score,
                        "video_path": metadata.get("video_path", "")
                    })
        
        # Process results for each video
        final_results = []

        for video_id, matches in video_results.items():
            if not matches:
                continue
                
            # Sort matches by score (desc)
            matches.sort(key=lambda x: x["score"], reverse=True)
            video_path = matches[0]["video_path"]

            timestamp_results = []
            for match in matches:
                ts = match.get("timestamp", 0)
                score = match.get("score", 0)
                timestamp_results.append(TimestampRange(
                    start=ts,
                    end=ts,
                    relevance_score=score
                ))

            max_score = timestamp_results[0].relevance_score if timestamp_results else 0

            final_results.append(VideoSearchResult(
                video_id=video_id,
                video_path=video_path,
                timestamps=timestamp_results,
                max_relevance_score=max_score
            ))
        
        # Sort videos by max relevance score
        final_results.sort(key=lambda x: x.max_relevance_score, reverse=True)
        
        # Limit to requested number of videos
        final_results = final_results[:request.top_k]
        
        return VideoSearchResponse(results=final_results)
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Unexpected error in /search_video: {e}")
        raise HTTPException(status_code=500, detail="Internal server error processing video search.")

@app.get("/videos", response_model=VideoListResponse)
async def list_videos():
    """
    List all indexed videos.
    """
    try:
        # Query all video frames to get unique video IDs
        results = collection.query(
            query_embeddings=[embed_text("video")],  # Dummy query
            n_results=10000,  # Large number to get all results
            include=["metadatas"],
            where={"type": "video_frame"}
        )
        
        video_ids = set()
        metadatas = results["metadatas"][0] if results["metadatas"] else []
        
        for metadata in metadatas:
            if metadata and "video_id" in metadata:
                video_ids.add(metadata["video_id"])
        
        return VideoListResponse(videos=sorted(list(video_ids)))
        
    except Exception as e:
        logger.error(f"Error listing videos: {e}")
        raise HTTPException(status_code=500, detail="Internal server error listing videos.")
