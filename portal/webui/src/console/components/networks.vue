<template>
  <el-main style="padding: 0; margin-bottom: 3em;">
    <div class="network-title">
        <h2>
          <i class="el-icon-user" style="margin-right: 0.5em"></i>
          <span>Networks</span>
        </h2>
        <el-button type="primary" @click="createNetwork" plain>
          <i class="el-icon-money" style="margin-right: 0.5em"></i>
          <span>Create Network</span>
        </el-button>
    </div>

    <el-dialog title="Create network" width="30%" :visible="showCreateNetwork" @close="showCreateNetwork = false">
      <el-form :model="networkOption">
        <el-form-item label="Name">
          <el-input v-model="networkOption.name" autocomplete="off"></el-input>
        </el-form-item>
        <el-form-item label="Description">
          <el-input v-model="networkOption.desc" autocomplete="off"></el-input>
        </el-form-item>
      </el-form>
      <div style="display: flex; justify-content: flex-end;">
        <el-button type="primary" @click="confirmCreateNetwork">Create</el-button>
      </div>
    </el-dialog>

    <el-table :data="networks" style="width: 100%; margin-top: 1em">
      <el-table-column prop="name" label="NAME"></el-table-column>
      <el-table-column label="Role" >
        <template #default="props">
          <el-tag :type="props.row.role === 'owner' ? 'warning' : props.row.role === 'member' ? 'success' : ''">
            {{ props.row.role.toUpperCase() }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="member_count" label="Members"></el-table-column>
      <el-table-column prop="device_count" label="Devices"></el-table-column>
      <el-table-column prop="description" label="Description"></el-table-column>
      <el-table-column width="60">
        <template #default="props">
          <el-button icon="el-icon-arrow-right" size="small" @click="$router.push('/console/network/' + props.row.network_id)" plain round circle></el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-main>
</template>

<script>
import service from "@/api/service";

export default {
  name: 'network',
  data: function () {
    return {
      networks: [],
      showCreateNetwork: false,
      networkOption: {},
    }
  },
  mounted() {
    service.get("/api/v1/networks").then(res => {
      this.networks = res.data.networks
    })
  },
  methods: {
    createNetwork: function () {
      this.showCreateNetwork = true
    },
    confirmCreateNetwork: function () {
      let self = this;
      service.post("/api/v1/network", {name: this.networkOption.name, description: this.networkOption.desc}).then(res => {
        self.showCreateNetwork = false
        self.networks.push(res.data.network)
      })
    }
  }
}
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>

.network-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.el-icon-arrow-down {
  font-size: 1em;
}

</style>
