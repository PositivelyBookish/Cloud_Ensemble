document.getElementById('upload-form').addEventListener('submit', async function (event) {
    event.preventDefault(); // Prevent the default form submission
  
    const form = event.target;
    const formData = new FormData(form);
  
    try {
        const response = await fetch(form.action, {
            method: form.method,
            body: formData,
        });
  
        if (!response.ok) {
            throw new Error('Error uploading image');
        }
  
        const result = await response.json();
        const responseDiv = document.getElementById('response');
        responseDiv.innerHTML = `
            <h3>Prediction:</h3>
            <p>${result.prediction}</p>
            <p>Confidence: ${(result.confidence * 100).toFixed(2)}%</p>
        `;
    } catch (error) {
        console.error(error);
        document.getElementById('response').textContent = 'An error occurred while processing the image.';
    }
  });
  