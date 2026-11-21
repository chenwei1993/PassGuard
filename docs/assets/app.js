(() => {
  const nav = document.getElementById("nav");
  const menuBtn = document.getElementById("menu-btn");
  const navLinks = document.getElementById("nav-links");

  const onScroll = () => nav.classList.toggle("scrolled", window.scrollY > 8);
  onScroll();
  window.addEventListener("scroll", onScroll, { passive: true });

  if (menuBtn && navLinks) {
    const close = () => {
      navLinks.classList.remove("open");
      menuBtn.setAttribute("aria-expanded", "false");
      menuBtn.setAttribute("aria-label", "打开菜单");
    };
    menuBtn.addEventListener("click", () => {
      const open = navLinks.classList.toggle("open");
      menuBtn.setAttribute("aria-expanded", String(open));
      menuBtn.setAttribute("aria-label", open ? "关闭菜单" : "打开菜单");
    });
    navLinks.querySelectorAll("a").forEach((a) => a.addEventListener("click", close));
    document.addEventListener("click", (e) => {
      if (!navLinks.contains(e.target) && e.target !== menuBtn && !menuBtn.contains(e.target)) {
        close();
      }
    });
  }

  const shots = {
    "vault-light": {
      src: "assets/vault-light.png",
      alt: "守密 PassGuard 保险库界面，显示 GitHub 密码详情",
      caption: "保险库：搜索、文件夹、强度与收藏一目了然，右侧直接查看详情。",
    },
    generator: {
      src: "assets/generator.png",
      alt: "守密 PassGuard 密码生成器界面",
      caption: "密码生成器：随机、易记、PIN 三种模式，实时强度评分与破解时间。",
    },
    security: {
      src: "assets/security.png",
      alt: "守密 PassGuard 安全检查界面",
      caption: "安全检查：弱密码、重复、已泄露、过期，按优先级集中处理。",
    },
    backup: {
      src: "assets/backup.png",
      alt: "守密 PassGuard 本地备份界面",
      caption: "本地备份：一键加密备份，自动保留最近若干份。",
    },
    "vault-dark": {
      src: "assets/vault-dark.png",
      alt: "守密 PassGuard 深色模式保险库界面",
      caption: "深色模式：跟随设置切换，夜里看也不刺眼。",
    },
  };

  const shot = document.getElementById("shot");
  const caption = document.getElementById("shot-caption");
  const tabs = document.querySelectorAll(".tab");

  tabs.forEach((tab) => {
    tab.addEventListener("click", () => {
      const key = tab.dataset.shot;
      const data = shots[key];
      if (!shot || !data) return;
      tabs.forEach((t) => {
        t.classList.toggle("active", t === tab);
        t.setAttribute("aria-selected", String(t === tab));
      });
      shot.src = data.src;
      shot.alt = data.alt;
      if (caption) caption.textContent = data.caption;
    });
  });

  const reveal = document.querySelectorAll(".reveal");
  if ("IntersectionObserver" in window && !window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
    const io = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            entry.target.classList.add("in");
            io.unobserve(entry.target);
          }
        });
      },
      { rootMargin: "0px 0px -8% 0px", threshold: 0.08 }
    );
    reveal.forEach((el) => io.observe(el));
  } else {
    reveal.forEach((el) => el.classList.add("in"));
  }

  const year = document.getElementById("year");
  if (year) year.textContent = String(new Date().getFullYear());
})();
