<template>
  <div class="login-container">
    <loading></loading>
    <span v-if="error === ''" class="login-loading-text">Login successfully, redirecting...</span>
    <span v-if="error !== ''" class="auth-error">{{ error }}</span>
  </div>
</template>

<script>
import Loading from '@/components/loading'
import * as login from "@/api/login";

export default {
  components:{Loading},
  data() {
    return {
      error: '',
    };
  },
  methods: {},
  mounted: function () {
    login.authCallback("github", window.location.search)
        .then(response => {
          localStorage.setItem('accessToken', response.data.access_token)
          if (response.data.notify_client !== 0) {
            const port = response.data.notify_client;
            const token = response.data.access_token;
            window.location.replace('http://localhost:' + port + '/local/auth/callback?token=' + token);
          } else {
            window.location.replace('/console');
          }
        })
        .catch(err => {
          console.log('Auth callback request failed', err)
          if(err.data !== null && err.data.error !== ''){
            this.error = err.data.error
          } else {
            this.error = err.statusText
          }
        })
  }
}
</script>

<style>
.auth-error {
  color: #f55252;
}
</style>