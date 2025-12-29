async function search() {
  const phrase = document.getElementById("searchInput").value.trim();
  if (!phrase) return;

  const results = document.getElementById("results");
  results.innerHTML = '<div class="loading">Searching...</div>';

  try {
    const response = await fetch(
      `/api/search?phrase=${encodeURIComponent(phrase)}`
    );
    if (!response.ok) throw new Error("Search failed");

    const data = await response.json();

    if (data.comics && data.comics.length > 0) {
      results.innerHTML = data.comics
        .map(
          (comic) => `
                <div class="comic" onclick="openImage(this.querySelector('img'))">
                    <img src="${comic.url}" alt="${comic.title || "Comic"}" loading="eager" />
                    <h3>${comic.title || `Comic #${comic.id}`}</h3>
                </div>
            `
        )
        .join("");
      
      data.comics.forEach(comic => {
        const img = new Image();
        img.src = comic.url;
      });
    } else {
      results.innerHTML = '<div class="error">No comics found</div>';
    }
  } catch (error) {
    results.innerHTML = '<div class="error">Error: ' + error.message + "</div>";
  }
}

function openImage(img) {
  let overlay = document.querySelector('.overlay');
  let closeBtn = document.querySelector('.close-btn');
  
  if (!overlay) {
    overlay = document.createElement('div');
    overlay.className = 'overlay';
    overlay.onclick = closeImage;
    document.body.appendChild(overlay);
  }
  
  if (!closeBtn) {
    closeBtn = document.createElement('button');
    closeBtn.className = 'close-btn';
    closeBtn.innerHTML = '×';
    closeBtn.onclick = closeImage;
    document.body.appendChild(closeBtn);
  }
  
  // Клонируем картинку и добавляем в body ПОСЛЕ overlay
  const clonedImg = img.cloneNode(true);
  clonedImg.id = 'expanded-image';
  document.body.appendChild(clonedImg);
  
  overlay.classList.add('active');
  closeBtn.classList.add('active');
}

function closeImage() {
  const img = document.getElementById('expanded-image');
  const overlay = document.querySelector('.overlay');
  const closeBtn = document.querySelector('.close-btn');
  
  if (img) img.remove();
  if (overlay) overlay.classList.remove('active');
  if (closeBtn) closeBtn.classList.remove('active');
}

document.getElementById("searchInput").addEventListener("keypress", (e) => {
  if (e.key === "Enter") search();
});

document.addEventListener("keydown", (e) => {
  if (e.key === "Escape") closeImage();
});
