import request from './request'

/**
 * 群组 API 服务层
 * 封装所有群组相关的HTTP请求
 */

// 获取用户所在的所有群组
export const getUserGroups = () => {
  return request.get('/groups')
}

// 创建群组
export const createGroup = (groupData) => {
  return request.post('/groups', groupData)
}

// 获取特定群组的详细信息
export const getGroupDetails = (groupId) => {
  return request.get(`/groups/${groupId}`)
}

// 更新群组信息
export const updateGroup = (groupId, updateData) => {
  return request.put(`/groups/${groupId}`, updateData)
}

// 解散群组
export const deleteGroup = (groupId) => {
  return request.delete(`/groups/${groupId}`)
}

// 获取群成员列表
export const getGroupMembers = (groupId) => {
  return request.get(`/groups/${groupId}/members`)
}

// 添加群成员
export const addGroupMember = (groupId, memberData) => {
  return request.post(`/groups/${groupId}/members`, memberData)
}

// 移除群成员
export const removeGroupMember = (groupId, userId) => {
  return request.delete(`/groups/${groupId}/members/${userId}`)
}

// 获取群组消息
export const getGroupMessages = (groupId, page = 1, pageSize = 50) => {
  return request.get(`/groups/${groupId}/messages?page=${page}&pageSize=${pageSize}`)
}

// 获取群在线成员
export const getOnlineMembers = (groupId) => {
  return request.get(`/groups/${groupId}/online-members`)
}

// 按用户名搜索用户
export const searchUsers = (keyword) => {
  return request.get('/users/search', { params: { keyword } })
}

// 修改成员角色
export const updateMemberRole = (groupId, userId, role) => {
  return request.put(`/groups/${groupId}/members/${userId}/role`, { role })
}

// 转让群主
export const transferGroupOwnership = (groupId, targetUserId) => {
  return request.post(`/groups/${groupId}/transfer-owner`, { target_user_id: targetUserId })
}