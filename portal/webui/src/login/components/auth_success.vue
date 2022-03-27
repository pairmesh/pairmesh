<template>
  <div class="login-container">
    <div class="wrapper"> <svg class="checkmark" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 52 52">
      <circle class="checkmark__circle" cx="26" cy="26" r="25" fill="none" />
      <path class="checkmark__check" fill="none" d="M14.1 27.2l7.1 7.2 16.7-16.8" />
    </svg>
    </div>
    <span class="login-loading-text">Success (check status by opening tray icon)</span>
  </div>
</template>

<script>
export default {
  data() {
    return {
      status: 'Success',
      error: '',
    }
  }
}
</script>

<style>
.wrapper {
  display: flex;
  justify-content: center;
  align-items: center;
  margin: 30px auto;
}

.checkmark__circle {
  stroke-dasharray: 166;
  stroke-dashoffset: 166;
  stroke-width: 2;
  stroke-miterlimit: 10;
  stroke: #6e90c4;
  fill: none;
  animation: stroke 0.6s cubic-bezier(0.65, 0, 0.45, 1) forwards
}

.checkmark {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  display: block;
  stroke-width: 2;
  stroke: #fff;
  stroke-miterlimit: 10;
  margin: 10% auto;
  box-shadow: inset 0px 0px 0px #6e90c4;
  animation: fill .4s ease-in-out .4s forwards, scale .3s ease-in-out .9s both
}

.checkmark__check {
  transform-origin: 50% 50%;
  stroke-dasharray: 48;
  stroke-dashoffset: 48;
  animation: stroke 0.3s cubic-bezier(0.65, 0, 0.45, 1) 0.8s forwards
}

@keyframes stroke {
  100% {
    stroke-dashoffset: 0
  }
}

@keyframes scale {

  0%,
  100% {
    transform: none
  }

  50% {
    transform: scale3d(1.1, 1.1, 1)
  }
}

@keyframes fill {
  100% {
    box-shadow: inset 0px 0px 0px 40px #6e90c4
  }
}
</style>