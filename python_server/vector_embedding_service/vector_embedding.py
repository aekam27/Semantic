import json
from vector_embedding_service.models import model

class VectorController:

    def generate_embeddings(self, rawText: str) -> dict:
        embeddings = model.encode(rawText)
        return {"vector_embedding":embeddings.tolist()}