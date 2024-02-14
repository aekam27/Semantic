from fastapi import APIRouter, Depends
from vector_embedding_service.vector_embedding import VectorController
from vector_embedding_service.payload_dtos import Payload

router = APIRouter()


@router.post("/vectorEmbeddings/")
async def generate_embeddings(payload: Payload, vector_controller: VectorController = Depends()):
    rawText = payload.name
    return vector_controller.generate_embeddings(rawText)