<template>
  <div class="login-container">
    <div class="login-options">
      <ul>
        <li v-for="(sso, index) in ssoOptions" :key="index" @click="login(sso.link)">
          <img src="/images/login/github.png" v-if="sso.name.toLowerCase() === 'github'">
          <span>{{ sso.name }} Login</span>
        </li>
      </ul>
    </div>
  </div>
</template>

<script>

import {ssoMethods} from '@/api/login'

export default {
  data() {
    return {
      ssoOptions: [],
    };
  },
  mounted() {
    ssoMethods(window.location.search).then(r => this.ssoOptions = r.data)
  },
  methods: {
    login: function (link) {
      window.location.href = '/login/auth/redirect?url='+encodeURIComponent(link)
    }
  }
}
</script>

<style>

.login-options {
  margin-top: 80px;
}

.login-options ul {
  list-style: none;
  padding: 0;
  margin: 0;
}

.login-options ul li {
  line-height: 40px;
  width: 300px;
  margin-bottom: 10px;
  border-radius: 5px;
  box-shadow: 0 2px 4px rgb(0 0 0 / 5%);
  border: 1px solid #d8d6d4;
  transition: border-color 200ms ease, box-shadow 200ms ease;
  display: flex;
  flex-direction: row;
  justify-content: center;
  align-items: center;
}

.login-options ul li:hover {
  border: 1px solid #aaa;
  cursor: pointer;
}

.login-options ul li img {
  height: 28px;
  width: 28px;
  margin-right: 10px;
}

.login-notes {
  max-width: 400px;
  margin-top: 100px;
  padding: 1.5em;
  font-size: 0.9em;
  color: #888;
  text-align: center;
}

.login-notes a {
  color: #777;
}

</style>