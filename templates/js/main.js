// Main JavaScript functionality

function copyCloneUrl(btn) {
  const url = btn.getAttribute("data-clone-url");
  navigator.clipboard.writeText(url).then(() => {
    btn.textContent = "Copied!";
    btn.style.color = "#28a745";
    setTimeout(() => {
      btn.textContent = "Copy";
      btn.style.color = "";
    }, 2000);
  });
}
