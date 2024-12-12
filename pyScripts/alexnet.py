# -*- coding: utf-8 -*-
"""Alexnet"""

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
IMAGE_SIZE = 80
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

# Ensure the model file exists
model_path = './models/AlexNet-model-80.pth'  # Update this path to your actual model file location
if not os.path.exists(model_path):
    print("Model file not found:", model_path)
    exit(1)

# Load the AlexNet model
model = models.alexnet(weights="DEFAULT")
model.classifier[6] = nn.Linear(4096, 15)
state_dict = torch.load(model_path, weights_only=True, map_location=torch.device('cpu'))
del state_dict['classifier.6.weight']
del state_dict['classifier.6.bias']
model.load_state_dict(state_dict, strict=False)
model.eval()

# Define transformations
transform = transforms.Compose([
    transforms.Resize((80, 80)),  # Resize to match the model's input size
    transforms.ToTensor(),       # Convert to a tensor
    transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225]),  # Normalize using ImageNet stats
])

# Check if the image path is provided
if len(sys.argv) < 2:
    print("Error: Image path not provided.")
    exit(1)

# Load the image from the provided path
image_path = sys.argv[1]  # Get the image path from command-line arguments
if not os.path.exists(image_path):
    print(f"Error: Image file not found at {image_path}")
    exit(1)

# Process the image
image = Image.open(image_path).convert("RGB")
input_tensor = transform(image).unsqueeze(0)  # Add batch dimension

# Ensure the model is in evaluation mode
model.eval()

# Perform the forward pass to get predictions
with torch.no_grad():  # No gradients needed during inference
    output = model(input_tensor)

# Get the predicted class and confidence score
_, predicted_class = torch.max(output, 1)  # Get the index of the highest score
confidence = torch.softmax(output, dim=1)[0, predicted_class].item()  # Confidence score

# Get the predicted class name
predicted_label = classification_types[predicted_class.item()]

# Print the result
print(f"Predicted Class: {predicted_label}")
print(f"Confidence: {confidence:.2%}")

# Measure the end time and print the duration
end_time = time.time()
execution_time = end_time - start_time
print(f"Execution Time: {execution_time:.2}")