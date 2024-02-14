from pydantic import BaseModel

class Payload(BaseModel):
    name: str