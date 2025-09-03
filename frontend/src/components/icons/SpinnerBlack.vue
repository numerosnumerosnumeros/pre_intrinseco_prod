<template>
  <svg
    width="30"
    height="14"
    viewBox="0 0 45 24"
    xmlns="http://www.w3.org/2000/svg"
  >
    <circle class="spinner_S1WN" cx="6" cy="12" r="6" />
    <circle class="spinner_S1WN spinner_Km9P" cx="22.5" cy="12" r="6" />
    <circle class="spinner_S1WN spinner_JApP" cx="39" cy="12" r="6" />
  </svg>
</template>

<style scoped>
.spinner_S1WN {
  animation: spinner_MGfb 0.8s linear infinite;
  animation-delay: -0.8s;
  fill: black;
}
.spinner_Km9P {
  animation-delay: -0.65s;
}
.spinner_JApP {
  animation-delay: -0.5s;
}
@keyframes spinner_MGfb {
  93.75%,
  100% {
    opacity: 0.2;
  }
}
</style>
