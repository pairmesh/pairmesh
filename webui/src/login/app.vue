<template>
  <div class="login-wrapper">
    <div style="display: flex; flex-direction: row;align-items: center;">
      <a href="/"><img src="/images/logo.svg" height="40px" alt=""/></a>
    </div>
    <router-view></router-view>
  </div>
</template>

<style>
.login-wrapper {
  bottom: 0;
  top: 0;
  position: absolute;
  right: 0;
  left: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 0;
  margin: 0;
}

.login-container {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.login-loading-text {
  font-size: 14px;
  color: #999;
}

</style>