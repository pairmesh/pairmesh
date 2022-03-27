<template>
  <el-row :gutter="10" type="flex" justify="center" style="margin: 0 0.7em">
    <el-col :xs="24" :sm="21" :md="17" :lg="14" :xl="12" style="display: flex">
      <slot></slot>
    </el-col>
  </el-row>
</template>

<script>
export default {
  name: 'flexbox',
  props: {}
}
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
</style>
