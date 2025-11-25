import os
import io
import uuid
from typing import List, Optional
import logging

from fastapi import FastAPI, UploadFile, File, HTTPException
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
# Use environment variable for path, default to local relative path
CHROMA_DB_PATH = os.getenv("CHROMA_DB_PATH", "chroma_data")
COLLECTION_NAME = "images"

# Pydantic Models
class IndexImageResponse(BaseModel):
    id: str

class ImageIndexResult(BaseModel):
    filename: str
    id: Optional[str] = None
    status: str
    error: Optional[str] = None

class BatchIndexResponse(BaseModel):
    results: List[ImageIndexResult]

class SearchRequest(BaseModel):
    query: str
    top_k: int = 5

class SearchResultItem(BaseModel):
    id: str
    score: Optional[float] = None
    metadata: Optional[dict] = None

class SearchResponse(BaseModel):
    results: List[SearchResultItem]

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
        collection = chroma_client.get_or_create_collection(name=COLLECTION_NAME)
    except Exception as e:
        logger.error(f"Failed to initialize ChromaDB: {e}")
        raise RuntimeError("Could not initialize vector database.")
        
    logger.info("Service startup complete.")

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

# Endpoints
@app.post("/index_image", response_model=IndexImageResponse)
async def index_image(file: UploadFile = File(...)):
    """
    Accepts an image file, embeds it, and stores it in the vector database.
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
        metadata = {
            "filename": file.filename,
            "content_type": file.content_type or "unknown"
        }

        # Store in Chroma
        collection.add(
            ids=[image_id],
            embeddings=[embedding],
            metadatas=[metadata]
        )
        
        return IndexImageResponse(id=image_id)

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Unexpected error in /index_image: {e}")
        raise HTTPException(status_code=500, detail="Internal server error processing image.")

@app.post("/index_batch", response_model=BatchIndexResponse)
async def index_batch(files: List[UploadFile] = File(...)):
    """
    Accepts multiple image files, embeds them, and stores them in the vector database.
    Returns results for each file.
    """
    if not files:
        raise HTTPException(status_code=400, detail="No files provided.")

    results = []
    
    # Prepare lists for batch adding to Chroma
    ids_to_add = []
    embeddings_to_add = []
    metadatas_to_add = []

    for file in files:
        result_item = ImageIndexResult(filename=file.filename or "unknown", status="pending")
        
        try:
            contents = await file.read()
            if not contents:
                result_item.status = "error"
                result_item.error = "Empty file"
                results.append(result_item)
                continue

            try:
                embedding = embed_image(contents)
            except ValueError:
                result_item.status = "error"
                result_item.error = "Invalid image format"
                results.append(result_item)
                continue
            
            # Success so far
            image_id = str(uuid.uuid4())
            metadata = {
                "filename": file.filename,
                "content_type": file.content_type or "unknown"
            }

            ids_to_add.append(image_id)
            embeddings_to_add.append(embedding)
            metadatas_to_add.append(metadata)

            result_item.id = image_id
            result_item.status = "success"
            results.append(result_item)
            
        except Exception as e:
            logger.error(f"Error processing file {file.filename}: {e}")
            result_item.status = "error"
            result_item.error = str(e)
            results.append(result_item)

    # Batch add to Chroma if there are any successful items
    if ids_to_add:
        try:
            collection.add(
                ids=ids_to_add,
                embeddings=embeddings_to_add,
                metadatas=metadatas_to_add
            )
        except Exception as e:
            logger.error(f"Error adding batch to Chroma: {e}")
            # If batch add fails, mark all 'success' items as failed
            for res in results:
                if res.status == "success":
                    res.status = "error"
                    res.error = "Database insertion failed"
            raise HTTPException(status_code=500, detail="Internal server error storing embeddings.")

    return BatchIndexResponse(results=results)

@app.post("/search", response_model=SearchResponse)
async def search(request: SearchRequest):
    """
    Accepts a natural language query, converts it to an embedding, 
    and returns the most similar images.
    """
    if not request.query.strip():
        raise HTTPException(status_code=400, detail="Query cannot be empty.")

    try:
        # Embed query
        query_embedding = embed_text(request.query)

        # Search Chroma
        results = collection.query(
            query_embeddings=[query_embedding],
            n_results=request.top_k,
            include=["metadatas", "distances"]
        )

        # Format response
        # Chroma results are lists of lists (one list per query). We only have one query.
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
        logger.error(f"Unexpected error in /search: {e}")
        raise HTTPException(status_code=500, detail="Internal server error processing search.")
