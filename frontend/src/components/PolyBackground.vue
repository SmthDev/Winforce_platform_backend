<script setup lang="ts">

</script>

<template>
  <div class="poly" aria-hidden="true">
    <img class="poly__mesh poly__mesh--top" src="/images/poly-mesh.svg" alt="" />
    <img class="poly__mesh poly__mesh--bottom" src="/images/poly-mesh.svg" alt="" />
  </div>
</template>

<style scoped>
.poly {
  position: absolute;
  inset: 0;
  overflow: hidden;
  pointer-events: none;
  z-index: 0;
}

.poly__mesh {
  position: absolute;
  height: auto;
}

.poly__mesh--top {
  top: -120px;
  right: -260px;
  width: 1100px;
  opacity: 0.5;
}

.poly__mesh--bottom {
  bottom: -260px;
  left: -360px;
  width: 900px;
  opacity: 0.35;
  transform: rotate(180deg);
}

@media (max-width: 720px) {
  .poly__mesh--top {
    top: -80px;
    right: -220px;
    width: 760px;
  }

  .poly__mesh--bottom {
    bottom: -200px;
    left: -280px;
    width: 640px;
  }
}
</style>
