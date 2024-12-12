import requests
from flask import Flask, request, jsonify, render_template
from werkzeug.utils import secure_filename
import uuid
import os

app = Flask(__name__)

@app.route('/')
def home():
    return render_template('index.html') 

@app.route('/classify', methods=['POST'])
def classify():
    try:
        # Get the image from the form
        image = request.files['image']
        if image:
            # Generate a unique filename for the image
            filename = str(uuid.uuid4()) + "___" + secure_filename(image.filename)

            # Send the image to the Nginx server running on port 8080
            nginx_url = 'http://localhost:8080/upload'  # Replace with your Nginx endpoint if different

            # Prepare the file for sending
            files = {'file': (filename, image.stream, image.content_type)}

            # Make a POST request to the Nginx server
            response = requests.post(nginx_url, files=files)
            if response.status_code == 200:
                return "Image forwarded to Nginx successfully"
            else:
                return f"Failed to forward image to Nginx, status code: {response.status_code}", 500
        else:
            return "No image uploaded", 400

    except Exception as e:
        print(f"Error: {str(e)}")
        return f"An error occurred: {str(e)}", 500


if __name__ == '__main__':
    app.run(debug=True, host="0.0.0.0", port=5000)
