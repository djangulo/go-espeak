async function downloadWav() {
  let response = await fetch('/download', {
    method: 'POST',
    headers: {
      'Accept': 'application/json'
    },
    body: new FormData(document.getElementById("wav-form"))
  });

  let result = await response.json();
  window.open(result, "_blank")
}