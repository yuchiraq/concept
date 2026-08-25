const revealObserver = new IntersectionObserver(
  (entries) => {
    entries.forEach((entry) => {
      if (entry.isIntersecting) {
        entry.target.classList.add("is-visible");
      }
    });
  },
  { threshold: 0.16 }
);

document.querySelectorAll(".reveal").forEach((node) => {
  revealObserver.observe(node);
});

const mobileTabbar = document.querySelector(".mobile-tabbar");

if (mobileTabbar) {
  let lastScrollY = window.scrollY;
  let isTicking = false;

  const syncMobileTabbar = () => {
    const currentScrollY = window.scrollY;
    const isScrollingDown = currentScrollY > lastScrollY + 8;
    const isScrollingUp = currentScrollY < lastScrollY - 8;

    if (currentScrollY < 32 || isScrollingUp) {
      document.body.classList.remove("is-scrolling-down");
    } else if (isScrollingDown) {
      document.body.classList.add("is-scrolling-down");
    }

    lastScrollY = currentScrollY;
    isTicking = false;
  };

  window.addEventListener(
    "scroll",
    () => {
      if (window.innerWidth > 760) {
        document.body.classList.remove("is-scrolling-down");
        lastScrollY = window.scrollY;
        return;
      }

      if (isTicking) {
        return;
      }

      window.requestAnimationFrame(syncMobileTabbar);
      isTicking = true;
    },
    { passive: true }
  );

  window.addEventListener("resize", () => {
    if (window.innerWidth > 760) {
      document.body.classList.remove("is-scrolling-down");
    }

    lastScrollY = window.scrollY;
  });
}
