async function loadStats() {
  try {
    const response = await fetch("/api/admin/statistics");
    const contentType = response.headers.get("content-type");
    if (!contentType || !contentType.includes("application/json")) {
      window.location.href = "/login";
      return;
    }

    if (!response.ok) {
      window.location.href = "/login";
      return;
    }

    const data = await response.json();

    document.getElementById("comicsTotal").textContent =
      data.stats.comics_total;
    document.getElementById("comicsFetched").textContent =
      data.stats.comics_fetched;
    document.getElementById("wordsTotal").textContent = data.stats.words_total;
    document.getElementById("wordsUnique").textContent =
      data.stats.words_unique;

    const statusDiv = document.getElementById("status");
    statusDiv.textContent = `Status: ${data.status}`;
    statusDiv.className = "status " + data.status;
  } catch (error) {
    window.location.href = "/login";
  }
}

async function updateDB() {
  if (!confirm("Start database update?")) return;
  try {
    const response = await fetch("/api/admin/update", { method: "POST" });
    if (response.ok) {
      alert("Update started successfully");
      setTimeout(loadStats, 1000);
    } else if (response.status === 202) {
      alert("Update already in progress");
    } else {
      throw new Error("Update failed");
    }
  } catch (error) {
    alert("Error: " + error.message);
  }
}

async function dropDB() {
  if (
    !confirm(
      "Are you sure you want to DROP the database? This cannot be undone!"
    )
  )
    return;
  try {
    const response = await fetch("/api/admin/db", { method: "DELETE" });
    if (response.ok) {
      alert("Database dropped successfully");
      setTimeout(loadStats, 1000);
    } else {
      throw new Error("Drop failed");
    }
  } catch (error) {
    alert("Error: " + error.message);
  }
}

function logout() {
  document.cookie = "jwt_token=; Max-Age=0; path=/";
}

loadStats();
setInterval(loadStats, 5000);
