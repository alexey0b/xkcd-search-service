function togglePassword() {
  const input = document.getElementById("password");
  const toggle = document.querySelector(".toggle-password");
  if (input.type === "password") {
    input.type = "text";
    toggle.textContent = "🙈";
  } else {
    input.type = "password";
    toggle.textContent = "🙉";
  }
}

// Check if already logged in
function checkExistingToken() {
  const token = document.cookie
    .split("; ")
    .find((row) => row.startsWith("jwt_token="));
  if (token) {
    window.location.href = "/admin";
  }
}

checkExistingToken();

document.getElementById("loginForm").addEventListener("submit", async (e) => {
  e.preventDefault();

  const username = document.getElementById("username").value;
  const password = document.getElementById("password").value;
  const errorDiv = document.getElementById("error");

  try {
    const response = await fetch("/api/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: username, password: password }),
    });

    if (response.ok) {
      window.location.href = "/admin";
    } else {
      errorDiv.textContent = "Invalid credentials";
      errorDiv.style.display = "block";
    }
  } catch (error) {
    errorDiv.textContent = "Login failed: " + error.message;
    errorDiv.style.display = "block";
  }
});
