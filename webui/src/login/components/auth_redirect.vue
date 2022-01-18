<template>
  <div class="login-container">
    <loading></loading>
    <span class="login-loading-text">Redirecting to authentication website...</span>
  </div>
</template>

<script>
import Loading from '@/components/loading'

export default {
  components: {Loading},
  data() {
    return {
      status: 'Redirecting',
      error: '',
    }
  },
  mounted: function () {
    var url = new URL(window.location)
    var redirectTo = url.searchParams.get("url")
    if (redirectTo === null || redirectTo === '') {
      this.error = 'Wrong destination address'
      return
    }
    redirectTo = decodeURIComponent(redirectTo)
    console.log(redirectTo)
    window.location.href = redirectTo
  }
}
</script>
