# -*- coding: utf-8 -*-
"""Convnext-Tiny"""

import os
import sys
import warnings
import torch
from torch import nn
from torchvision import transforms, models
from PIL import Image
import time  # Import time module

warnings.filterwarnings("ignore")

# Constants and parameters
IMAGE_SIZE = 224  # ConvNeXt-Tiny default input size
classification_types = [
    'Pepper__bell___Bacterial_spot', 'Pepper__bell___healthy',
    'Potato___Early_blight', 'Potato___Late_blight', 'Potato___healthy',
    'Tomato_Bacterial_spot', 'Tomato_Early_blight', 'Tomato_Late_blight',
    'Tomato_Leaf_Mold', 'Tomato_Septoria_leaf_spot',
    'Tomato_Spider_mites_Two_spotted_spider_mite', 'Tomato__Target_Spot',
    'Tomato__Tomato_YellowLeaf__Curl_Virus', 'Tomato__Tomato_mosaic_virus',
    'Tomato_healthy'
]

# Measure the start time
start_time = time.time()

# Check if the image path is provided
if len(sys.argv) < 2:
    print("Error: Image path not provided.")
    exit(1)

# Load the image path from command-line arguments
image_path = sys.argv[1]
if not os.path.exists(image_path):
    print(f"Error: Image file not found at {image_path}")
    exit(1)

# Path to the pre-trained model
model_path = './models/convnext_tiny-model-89.pth'  # Update this path
if not os.path.exists(model_path):
    print("Error: Model file not found.")
    exit(1)

# Initialize ConvNeXt-Tiny
model = models.convnext_tiny(weights="DEFAULT")
model.classifier[2] = nn.Linear(model.classifier[2].in_features, len(classification_types))  # Modify final layer for 15 classes

# Load the pre-trained model state dict
state_dict = torch.load(model_path, weights_only=True, map_location=torch.device('cpu'))
if 'classifier.2.weight' in state_dict:
    del state_dict['classifier.2.weight']
if 'classifier.2.bias' in state_dict:
    del state_dict['classifier.2.bias']
model.load_state_dict(state_dict, strict=False)

# Freeze feature extraction layers
for param in model.features.parameters():
    param.requires_grad = False

# Set model to evaluation mode
model.eval()

# Image transformations for ConvNeXt
transform = transforms.Compose([
    transforms.Resize((IMAGE_SIZE, IMAGE_SIZE)),  # Resize to ConvNeXt's input size
    transforms.ToTensor(),                       # Convert to tensor
    transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225]),  # ImageNet stats
])

# Prediction function for the model
def predict_image(model, image_path):
    """
    Predict the class of a single image.

    Parameters:
        model (nn.Module): The fine-tuned model.
        image_path (str): Path to the image.

    Returns:
        str: Predicted class name.
        float: Confidence score (percentage).
    """
    # Load image and transform
    image = Image.open(image_path).convert("RGB")
    input_tensor = transform(image).unsqueeze(0)  # Add batch dimension

    # Perform prediction
    model.eval()
    with torch.no_grad():
        output = model(input_tensor)

    # Get the predicted class and confidence
    _, predicted_class = torch.max(output, 1)
    confidence = torch.softmax(output, dim=1)[0, predicted_class].item()

    predicted_label = classification_types[predicted_class.item()]
    return predicted_label, confidence * 100

# Predict the class for the input image
predicted_label, confidence = predict_image(model, image_path)
print(f"Predicted Class: {predicted_label}")
print(f"Confidence: {confidence:.2f}%")

# Measure the end time and print the duration
end_time = time.time()
execution_time = end_time - start_time
print(f"Execution Time: {execution_time:.2f}")