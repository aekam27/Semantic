
from fastapi import FastAPI
from router.router import router as vector_router

app = FastAPI()
app.include_router(vector_router, prefix="/v1")