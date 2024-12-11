// Get the form and the file input element
const form = document.getElementById('upload-form');
const fileInput = document.getElementById('image');
const responseDiv = document.getElementById('response');

// Handle form submission
form.addEventListener('submit', async (event) => {
  event.preventDefault();  // Prevent the default form submission

  const file = fileInput.files[0];
  if (!file) {
    responseDiv.innerHTML = "Please select an image to upload.";
    return;
  }

  // Prepare the form data
  const formData = new FormData();
  formData.append("image", file);

  try {
    // Send the image to the Nginx server
    const response = await fetch('/classify', {
      method: 'POST',
      body: formData,
    });

    if (response.ok) {
      const result = await response.json();  // Assuming the Go server sends a JSON response
      responseDiv.innerHTML = `Prediction Result: ${result.prediction}`;  // Update with the result
    } else {
      responseDiv.innerHTML = "Error: Unable to classify the image.";
    }
  } catch (error) {
    console.error("Error during fetch:", error);
    responseDiv.innerHTML = "An error occurred while uploading the image.";
  }
});
