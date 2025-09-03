<script setup>
import { RouterView } from 'vue-router'
import HeaderLayout from '@components/layout/HeaderLayout.vue'
import FooterLayout from '@components/layout/FooterLayout.vue'
</script>

<template>
  <div class="layout">
    <HeaderLayout />
    <main>
      <RouterView />
    </main>
    <FooterLayout />
  </div>
</template>

<style scoped>
.layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

main {
  flex: 1;
}
</style>
