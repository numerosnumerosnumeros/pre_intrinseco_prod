<script setup>
import { tokenStateStore } from '@state/session.js'
import { useHead } from '@unhead/vue'

useHead({
  title: 'nodo.finance',
  meta: [
    {
      name: 'twitter:title',
      content: 'nodo.finance',
    },
    {
      property: 'og:title',
      content: 'nodo.finance',
    },
    {
      name: 'robots',
      content: 'index, follow',
    },
  ],
})
</script>

<template>
  <div class="wrapper">
    <div class="box">
      <h1>Data. Finance. Clarity.</h1>
      <p>
        Turn any financial report into actionable data. Upload files or URLs and
        let our AI extract key metrics instantly.
      </p>
      <div class="nav-link-wrapper">
        <RouterLink
          to="/app"
          class="landing-button"
          v-if="tokenStateStore.value !== 'MountedPremium'"
        >
          <p>Go Professional</p>
        </RouterLink>
        <RouterLink to="/documentation" class="landing-button" v-else>
          <p>Documentation &#8594;</p>
        </RouterLink>
      </div>

      <div style="padding: 56.25% 0 0 0; position: relative">
        <iframe
          src="https://player.vimeo.com/video/1093498580?badge=0&amp;autopause=0&amp;player_id=0&amp;app_id=58479&amp;autoplay=1&amp;muted=1&amp;loop=1&amp;background=1"
          frameborder="0"
          allow="autoplay; fullscreen; picture-in-picture; clipboard-write; encrypted-media; web-share"
          style="position: absolute; top: 0; left: 0; width: 100%; height: 100%"
          title="landing_demo"
        ></iframe>
      </div>
    </div>
  </div>
</template>

<style scoped>
.video-wrapper {
  box-shadow: none;
  border: none;
  box-sizing: border-box;
}
iframe {
  border-radius: 15px;
  width: 100%;
  height: 100%;
  box-shadow: none;
  border: none;
  box-sizing: border-box;
  clip-path: inset(0.5px 0);
  mask: linear-gradient(to bottom, black 80%, transparent 100%);
  -webkit-mask: linear-gradient(to bottom, black 80%, transparent 100%);
}
.nav-link-wrapper p {
  display: flex;
  justify-content: end;
  font-size: 22px;
  text-align: right;
  padding: 0;
  width: fit-content;
}

.landing-button {
  background-color: black;
  border-radius: 10px;
  padding: 5px 10px;
  text-align: center;
  text-decoration: none;
  color: white;
  margin: 0;
  margin-bottom: 10px;
}

.nav-link-wrapper {
  display: flex;
  justify-content: end;
}

h1 {
  font-size: 30px;
  font-weight: 500;
  margin: 0;
  text-align: left;
  letter-spacing: 0.8px;
}

p {
  font-size: 18px;
  margin: 0;
  text-align: left;
  letter-spacing: 0.5px;
}
.wrapper {
  display: flex;
  justify-content: center;
  align-items: top;
  gap: 50px;
  margin-top: 20px;
  margin-bottom: 30px;
  min-height: 400px;
}

.wrapper > div {
  background: linear-gradient(360deg, transparent 0%, var(--gray-one) 100%);
}

.box {
  width: 750px;
  min-height: 260px;
  display: flex;
  flex-direction: column;
  gap: 30px;
  padding: 35px 40px;
  border-radius: 40px;
}

.bottom-cta {
  margin: 10px 0 30px;
}

@media (max-width: 910px) {
  .box {
    width: 80%;
  }
  .landing-button p {
    font-size: 18px;
  }
}

@media (max-width: 580px) {
  .box {
    gap: 15px;
    padding: 25px;
    border-radius: 30px;
  }
}

@media (max-width: 480px) {
  .box {
    min-height: auto;
    padding: 20px;
    width: 85%;
  }
  p {
    font-size: 16px;
  }
  h1 {
    font-size: 28px;
  }
  .landing-button p {
    font-size: 16px;
  }
}

@media (max-width: 370px) {
  .box {
    padding: 15px;
  }
}
</style>
