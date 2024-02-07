from sentence_transformers import SentenceTransformer
import sys
import json
model = SentenceTransformer('sentence-transformers/all-MiniLM-L6-v2')
rawText = sys.argv[1]
def generate_embedding() -> str:
    embeddings = model.encode(rawText)
    json_string = json.dumps({"vector_embedding":embeddings.tolist()})
    return "cmd_op_str_trans_:"+json_string