async function fetchRanking() {
  try {
    const response = await fetch("http://localhost:8080/api/rankings"); // adjust port if needed
    const data = await response.json();

    const tbody = document.querySelector("#rankingTable tbody");
    tbody.innerHTML = ""; // clear old rows

    data.forEach((player, index) => {
      const row = document.createElement("tr");

      row.innerHTML = `
        <td>${index + 1}</td>
        <td>${player.Username}</td>
        <td>${player.Score}</td>
      `;

      tbody.appendChild(row);
    });
  } catch (error) {
    console.error("Error fetching ranking:", error);
  }
}

// Fetch rankings when page loads
document.addEventListener("DOMContentLoaded", fetchRanking);
