<script setup>
// import FaviconNews from '@icons/FaviconNews.vue'
// import TelegramIcon from '@icons/TelegramIcon.vue'
</script>

<template>
  <footer>
    <!-- <div class="social-media-icons">
      <a
        href="https://news.nodo.finance"
        target="_blank"
        rel="noopener noreferrer"
        class="block"
      >
        <FaviconNews />
      </a>
      <a
        href="https://t.me/nodofinance"
        target="_blank"
        rel="noopener noreferrer"
        class="block"
        aria-label="Follow us on Telegram"
      >
        <TelegramIcon />
      </a>
    </div> -->

    <RouterLink to="/documentation" class="block">Documentation</RouterLink>
    <RouterLink to="/terms-of-service" class="block"
      >Terms of service</RouterLink
    >
  </footer>
</template>

<style scoped>
footer {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin: 20px auto 80px;
  gap: 20px;
}

.block {
  display: inline-block;
  font-size: 16px;
  margin: 5px auto;
  padding: 0;
  letter-spacing: 1px;
  background-color: var(--gray-two);
  box-shadow: 0 0 5px var(--gray-two);
  border-radius: 10px;
}
.social-media-icons {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 30px;
}
</style>
