document.documentElement.classList.add("has-js");

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

const mobileNav = document.querySelector("[data-mobile-nav]");
const menuToggle = document.querySelector(".menu-toggle");
const navLinks = document.querySelectorAll(".topnav a, .topbar-cta");

if (mobileNav && menuToggle) {
  const closeMenu = () => {
    mobileNav.classList.remove("is-open");
    document.body.classList.remove("has-menu-open");
    menuToggle.setAttribute("aria-expanded", "false");
  };

  const openMenu = () => {
    mobileNav.classList.add("is-open");
    document.body.classList.add("has-menu-open");
    menuToggle.setAttribute("aria-expanded", "true");
  };

  menuToggle.addEventListener("click", () => {
    if (mobileNav.classList.contains("is-open")) {
      closeMenu();
      return;
    }

    openMenu();
  });

  navLinks.forEach((link) => {
    link.addEventListener("click", () => {
      closeMenu();
    });
  });

  window.addEventListener("resize", () => {
    if (window.innerWidth > 760) {
      closeMenu();
    }
  });

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeMenu();
    }
  });
}
