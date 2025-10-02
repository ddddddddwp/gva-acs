<template>
  <div class="layout-container">
    <router-view class="main-view" />
  </div>
</template>

<script setup>
// TR069设备管理主页面
</script>

<style lang="scss" scoped>
.layout-container {
  background-color: #f0f2f5;
  height: 100%;
  width: 100%;
  .main-view {
    padding: 12px;
    height: 100%;
  }
}
</style>