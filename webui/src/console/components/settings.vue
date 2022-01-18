<template>
  <el-main style="padding: 0; margin-bottom: 3em;">
    <div class="setting-panel">
      <div class="setting-panel-item">
        <h4>Name</h4>
        <el-input v-model="userProfile.name" placeholder="NAME"></el-input>
      </div>
      <div class="setting-panel-item">
        <h4>Email</h4>
        <el-input v-model="userProfile.email" placeholder="EMAIL"></el-input>
      </div>
      <div class="setting-panel-tips">
        <span>All of the fields on this page are optional and can be deleted at any time, and by filling them out, you're giving us consent to share this data wherever your user profile appears. Please see our privacy statement to learn more about how we use this information.</span>
      </div>
      <el-button type="primary" @click="updateUserProfile">Update Profile</el-button>
    </div>
  </el-main>
</template>

<script>
import service from '@/api/service'

export default {
  name: 'settings',
  data: function () {
    return {
      userProfile: {
        name: 'Loading',
        email: 'Loading',
      },
      teamProfile: {
        name: 'Loading'
      },
    }
  },
  mounted() {
    let self = this
    service.get('/api/v1/user/profile').then(res => {
      self.userProfile = res.data
    })
  },
  methods: {
    updateUserProfile: function () {
      let self = this
      service.put('/api/v1/settings/user/profile', {
        'name': this.userProfile.name,
        'email': this.userProfile.email,
      }).then(res => {
        self.userProfile = res.data
        self.$emit('update-profile', {
          name: self.userProfile.name,
          email: self.userProfile.email,
        })
        self.$message({
          message: 'Update user profile successfully',
          type: 'success',
        })
      })
    },
  }
}
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
.setting-panel {
  margin-left: 1em;
}

.setting-panel h4 {
  margin: 0;
  line-height: 2.4em;
}

.setting-panel-tips {
  margin: 1em 0;
  color: #777;
  font-size: 0.8em;
}

.setting-panel-item {
  margin-bottom: 1em;
}
</style>
