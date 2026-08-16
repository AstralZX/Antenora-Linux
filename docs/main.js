// Antenora — site behaviour: ember particles, scroll reveal, rotating quotes.

const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

// ---- Ember particles -------------------------------------------------------
const QUOTES = [
  ["Abandon all hope, ye who enter here.", "Inferno, Canto III"],
  ["In the middle of the journey of our life I found myself within a dark woods where the straight way was lost.", "Inferno, Canto I"],
  ["The hottest places in hell are reserved for those who, in times of great moral crisis, maintain their neutrality.", "Inferno, Canto III"],
  ["Consider your origin; you were not formed to live like brutes but to follow virtue and knowledge.", "Inferno, Canto XXVI"],
  ["There is no greater sorrow than to recall happiness in times of misery.", "Inferno, Canto V"],
  ["Follow your own star.", "Inferno, Canto XV"],
  ["The secret of getting things done is to act!", "Inferno, Canto XXIV"],
  ["Here must all distrust be left behind; all cowardice must be ended.", "Inferno, Canto III"],
  ["You shall find out how salt is the taste of another's bread.", "Paradiso, Canto XVII"],
  ["Do not be afraid; our fate cannot be taken from us; it is a gift.", "Inferno, Canto II"],
];

function startEmbers() {
  const canvas = document.getElementById("embers");
  if (!canvas || reduceMotion) return;
  const ctx = canvas.getContext("2d");
  let w, h;
  const resize = () => { w = canvas.width = canvas.offsetWidth; h = canvas.height = canvas.offsetHeight; };
  resize();
  window.addEventListener("resize", resize);

  const N = 90;
  const embers = Array.from({ length: N }, () => ({
    x: Math.random() * w,
    y: Math.random() * h,
    r: Math.random() * 2.2 + 0.4,
    vx: (Math.random() - 0.5) * 0.3,
    vy: -(Math.random() * 0.7 + 0.15),
    a: Math.random() * 0.6 + 0.15,
    hue: Math.random() < 0.75 ? 14 : 36, // red or ember
  }));

  function tick() {
    ctx.clearRect(0, 0, w, h);
    for (const e of embers) {
      e.x += e.vx; e.y += e.vy;
      e.vy -= 0.002; // slow, like drifting ash
      e.a -= 0.0012;
      if (e.y < -10 || e.a <= 0) { e.y = h + 10; e.x = Math.random() * w; e.a = Math.random() * 0.6 + 0.2; }
      const g = ctx.createRadialGradient(e.x, e.y, 0, e.x, e.y, e.r * 5);
      g.addColorStop(0, `hsla(${e.hue}, 85%, 55%, ${e.a})`);
      g.addColorStop(1, "transparent");
      ctx.fillStyle = g;
      ctx.beginPath();
      ctx.arc(e.x, e.y, e.r * 5, 0, Math.PI * 2);
      ctx.fill();
    }
    requestAnimationFrame(tick);
  }
  tick();
}

// ---- Scroll reveal ---------------------------------------------------------
function startReveal() {
  const els = document.querySelectorAll(".reveal");
  if (reduceMotion || !("IntersectionObserver" in window)) {
    els.forEach((el) => el.classList.add("in"));
    return;
  }
  const io = new IntersectionObserver((entries) => {
    entries.forEach((e) => { if (e.isIntersecting) { e.target.classList.add("in"); io.unobserve(e.target); } });
  }, { threshold: 0.12 });
  els.forEach((el) => io.observe(el));
}

// ---- Rotating quote --------------------------------------------------------
function startQuote() {
  const q = document.querySelector("[data-quote]");
  const c = document.querySelector("[data-quote-cite]");
  if (!q || !c) return;
  let i = 0;
  const next = () => {
    const [text, cite] = QUOTES[i % QUOTES.length];
    q.style.opacity = "0"; c.style.opacity = "0";
    setTimeout(() => { q.textContent = text; c.textContent = cite; q.style.opacity = "1"; c.style.opacity = "1"; }, 400);
    i++;
  };
  q.style.transition = "opacity .4s ease"; c.style.transition = "opacity .4s ease";
  next();
  setInterval(next, 6500);
}

document.addEventListener("DOMContentLoaded", () => {
  startEmbers();
  startReveal();
  startQuote();
});
