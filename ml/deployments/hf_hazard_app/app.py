import joblib
from sentence_transformers import SentenceTransformer
import numpy as np
import re
import os
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import uvicorn

# --- Define input and output data models ---
class TextInput(BaseModel):
    """The JSON input model. Expects: {"text": "..."}"""
    text: str

class PredictionOutput(BaseModel):
    """The JSON output model."""
    label: str
    probability_hazardous: float
    text: str

# --- Load Models ---
# Paths are relative to the app.py file
SBERT_PATH = "./model_artifacts/sbert_model"
LR_PATH = "./model_artifacts/lr_relevance.joblib"

print("Loading SentenceTransformer model...")
try:
    _sbert = SentenceTransformer(SBERT_PATH, device="cpu")
    print("SBERT model loaded successfully.")
except Exception as e:
    print(f"Error loading SBERT model: {e}")
    _sbert = None

print("Loading LogisticRegression model...")
try:
    _lr = joblib.load(LR_PATH)
    print("Classifier loaded successfully.")
except Exception as e:
    print(f"Error loading classifier: {e}")
    _lr = None

# --- Preprocessing Function (Copied from your notebook) ---
def clean_text(s: str) -> str:
    """Uses the exact same cleaning as the training script."""
    s = re.sub(r"http\S+|www\.\S+", " ", s)       # links
    s = re.sub(r"#(\w+)", r"\1", s)               # hashtags -> word
    s = re.sub(r"@\w+", " ", s)                   # mentions
    s = re.sub(r"\s+", " ", s).strip()
    return s

# --- Initialize FastAPI App ---
app = FastAPI(
    title="Hazard Detection API",
    description="An API to classify text as hazard-related or not.",
    version="1.0.0"
)

# --- Define a root endpoint (for health check) ---
@app.get("/")
def read_root():
    return {"message": "Hazard Detection API is running. POST to /predict"}

# --- Define the prediction endpoint ---
@app.post("/predict", response_model=PredictionOutput)
async def predict(item: TextInput):
    """
    Predicts if the text is hazard-related.
    """
    if not _sbert or not _lr:
        raise HTTPException(status_code=503, detail="Models are not loaded. Check server logs.")

    text = item.text
    if not text or not text.strip():
        return PredictionOutput(
            label="Not Hazardous", 
            probability_hazardous=0.0, 
            text=text or ""
        )

    # 1. Clean the text
    cleaned_text = clean_text(text)
    
    # 2. Get embedding
    emb = _sbert.encode(
        [cleaned_text], 
        convert_to_numpy=True, 
        normalize_embeddings=True
    )
    
    # 3. Get probability (class 1 is 'relevant/hazardous')
    prob_hazardous = float(_lr.predict_proba(emb)[0, 1])
    
    # 4. Format output
    if prob_hazardous > 0.5:
        label = "Hazardous"
    else:
        label = "Not Hazardous"
        
    return PredictionOutput(
        label=label, 
        probability_hazardous=prob_hazardous, 
        text=text
    )



# --- Add this to the VERY BOTTOM of your app.py ---



if __name__ == "__main__":

    port = int(os.environ.get("PORT", 7860))
    uvicorn.run(app, host="0.0.0.0", port=port)


