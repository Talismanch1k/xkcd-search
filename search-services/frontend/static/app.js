function observeLast() {
  const cards = document.querySelectorAll(".comic-card");
  if (cards.length === 0) return;

  const last = cards[cards.length - 1];
  const io = new IntersectionObserver(
    async ([entry]) => {
      if (!entry.isIntersecting) return;
      io.disconnect();

      try {
        const res = await fetch(`/feed/next?id=${last.dataset.id}`);
        if (!res.ok) return;

        const { comics } = await res.json();
        const feed = document.getElementById("feed");

        for (const c of comics) {
          const section = document.createElement("section");
          section.className = "comic-card";
          section.dataset.id = c.ID;
          section.innerHTML = `
                      <img src="${c.URL}" alt="${c.Alt ?? ""}" loading="lazy">
                      ${c.Title ? `<p class="comic-title">${c.Title}</p>` : ""}
                  `;
          feed.appendChild(section);
        }
        observeLast();
      } catch (err) {
        console.error("feed/next failed", err);
      }
    },
    { threshold: 0.8 },
  );

  io.observe(last);
}

document.addEventListener("DOMContentLoaded", observeLast);
