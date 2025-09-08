<template>
  <div class="login-page">
    <el-card class="box-card">
      <div slot="header" class="clearfix">
        <span class="login-title">CHAT聊天室</span>
      </div>
      <div class="login-form">
        <el-form :model="form" :rules="loginRules" ref="loginForm">
          <el-form-item prop="userName">
            <el-input
              type="text"
              v-model="form.userName"
              auto-complete="off"
              placeholder="请输入用户名"
            >
              <template #prepend>
                <i style="font-size:20px" class="el-icon-user"></i>
              </template>
            </el-input>
          </el-form-item>

          <el-form-item prop="passWord">
            <el-input
              type="password"
              v-model="form.passWord"
              auto-complete="off"
              placeholder="请输入密码"
            >
              <template #prepend>
                <i style="font-size:20px" class="el-icon-key"></i>
              </template>
            </el-input>
          </el-form-item>

          <el-form-item>
            <el-button
              style="width:100%;"
              type="primary"
              @click="handleLogin"
              :loading="loading"
            >
              登录
            </el-button>
          </el-form-item>

          <el-form-item>
            <el-button
              style="width:100%;"
              type="primary"
              @click="handleRegister"
              :loading="loading"
            >
              注册
            </el-button>
          </el-form-item>
        </el-form>
      </div>
    </el-card>
  </div>
</template>

<script>
import request from '@/utils/request'

export default {
  name: 'Login',
  data() {
    return {
      loading: false,
      form: { userName: '', passWord: '' },
      loginRules: {
        userName: [{ required: true, message: '请输入账户', trigger: 'blur' }],
        passWord: [{ required: true, message: '请输入密码', trigger: 'blur' }]
      }
    }
  },
  methods: {
    handleLogin() {
      this.$refs.loginForm.validate().then(() => {
        this.loading = true
        request.post('/login', {
          username: this.form.userName,
          password: this.form.passWord
        }).then(res => {
          this.loading = false
          localStorage.setItem('token', res.token)
          localStorage.setItem('username', res.username)
          this.$router.push('/chat')
        }).catch(err => {
          this.loading = false
          this.$message.error(err?.error || '登录失败')
        })
      }).catch(() => {
        this.$message.warning('请填写完整信息')
      })
    },
    handleRegister() {
      this.$refs.loginForm.validate().then(() => {
        this.loading = true
        request.post('/register', {
          username: this.form.userName,
          password: this.form.passWord
        }).then(() => {
          this.loading = false
          this.$message.success('注册成功，请登录')
        }).catch(err => {
          this.loading = false
          this.$message.error(err?.error || '注册失败')
        })
      }).catch(() => {
        this.$message.warning('请填写完整信息')
      })
    }
  }
}
</script>


<style scoped>
.login-page {
  background-image: linear-gradient(180deg, #2af598 0%, #009efd 100%);
  height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
}

.login-title {
  font-size: 20px;
}

.box-card {
  width: 375px;
}
</style>
