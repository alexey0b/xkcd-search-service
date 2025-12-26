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
                <div class="comic">
                    <img src="${comic.url}" alt="${comic.title || "Comic"}" />
                    <h3>${comic.title || `Comic #${comic.id}`}</h3>
                </div>
            `
        )
        .join("");
    } else {
      results.innerHTML = '<div class="error">No comics found</div>';
    }
  } catch (error) {
    results.innerHTML = '<div class="error">Error: ' + error.message + "</div>";
  }
}

document.getElementById("searchInput").addEventListener("keypress", (e) => {
  if (e.key === "Enter") search();
});
