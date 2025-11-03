// Main JavaScript functionality

function copyCloneUrl(event, link) {
  event.preventDefault();
  const url = link.getAttribute("data-clone-url");
  navigator.clipboard.writeText(url).then(() => {
    const originalColor = link.style.color;
    link.style.color = "#38a169";
    setTimeout(() => {
      link.style.color = originalColor;
    }, 1000);
  });
}
