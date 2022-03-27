<template>
  <div>
    <!-- Header -->
    <div style="border-bottom: 1px solid #f1eded; margin: 1em 0;">
      <flexbox>
        <div style="display: flex; align-content: center; align-items: center;">
          <a href="/"><img src="/images/logo.svg" style="height: 30px; margin: 1em 1em 1em 0;" alt=""/></a>
        </div>
        <div class="header-support">
          <el-button v-popover:profile type="text" size="mini" class="profile-btn">
            <img width="30px" v-bind:src="userProfile.avatar" alt="">
            <span style="font-size: 1.5em; color: #3a74fa;margin-left: 0.5em"> {{ userProfile.name }}</span>
            <i class="el-icon-arrow-down" style="font-size: 1.2em;color: #3a74fa; margin-left: 0.2em;"></i>
          </el-button>
          <el-popover
              ref="profile"
              placement="bottom-end"
              width="250">
            <div class="profile-link">
              <el-link>Logout</el-link>
            </div>
          </el-popover>
        </div>
      </flexbox>

      <!-- Body -->
      <flexbox>
        <el-menu :default-active="activeIndex"
                 mode="horizontal"
                 @select="handleSelect"
                 active-text-color="#3a74fa"
                 style="border-bottom: none">
          <el-menu-item v-for="(item, index) in menuItems"
                        v-bind:key="index" :index="item.name"
                        style="font-size: 1em; font-weight: 500">
            <i :class="item.icon"></i>
            <span>{{ item.name.charAt(0).toUpperCase() + item.name.slice(1) }}</span>
          </el-menu-item>
        </el-menu>
      </flexbox>
    </div>

    <div>
      <flexbox>
        <router-view></router-view>
      </flexbox>
    </div>
  </div>
</template>

<script>

import Flexbox from "@/console/components/flexbox";
import service from "@/api/service";

export default {
  components: {Flexbox},
  data() {
    return {
      menus: [
        {
          name: 'overview',
          icon: 'el-icon-menu',
        },
        {
          name: 'networks',
          icon: 'el-icon-user-solid',
        },
        {
          name: 'keys',
          icon: 'el-icon-s-finance',
        },
        {
          name: 'settings',
          icon: 'el-icon-s-tools',
        }
      ],
      activeIndex: 'overview',
      fromClient: 0,
      userProfile: {
        name: 'Loading',
        email: 'Loading'
      },
    };
  },
  computed: {
    menuItems: function () {
      if (this.plan === 'free') {
        return this.menus.filter(el => el.name.toLowerCase() !== 'team')
      } else {
        return this.menus
      }
    },
  },
  mounted() {
    const parts = this.$route.path.split("/")
    if (parts.length < 3 || parts[1] !== 'console') { // start with /console/xxx
      this.$router.push('/console/overview')
    } else {
      this.activeIndex = parts[2]
    }

    let self = this
    service.get('/api/v1/user/profile').then(res => {
      self.userProfile = res.data
    })
  },
  methods: {
    handleSelect(key) {
      this.activeIndex = key
      this.$router.push('/console/' + key.toLowerCase())
    },
  },
}
</script>

<style>

body {
  padding: 0;
  margin: 0;
  overflow-x: hidden;
}

.header-support {
  margin-left: auto;
  display: flex;
  align-items: center;
  font-size: 0.9em;
}

.header-support a {
  margin-right: 1em;
  color: #656568;
  line-height: 1.1em;
  text-decoration: none;
}

.header-support a:hover {
  color: #444;
}

.profile-menu-label div {
  padding: 0.1em 0;
}

.profile-link {
  padding: 0.5em;
}

.profile-btn {
  display: flex;
  align-items: center;
}

.profile-btn span {
  display: flex;
  align-items: center;
}

</style>
