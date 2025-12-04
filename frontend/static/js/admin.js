// SwiftPost 管理员面板 JavaScript

class AdminPanel {
    constructor() {
        this.token = localStorage.getItem('token');
        this.user = JSON.parse(localStorage.getItem('user') || '{}');
        this.usersTable = null;
        this.emailsTable = null;
        
        this.init();
    }
    
    init() {
        this.checkAdminAuth();
        this.bindEvents();
        this.loadDashboardStats();
        this.loadUsers();
        this.setupDataTables();
        
        // 设置标签页切换
        this.setupTabSwitching();
    }
    
    checkAdminAuth() {
        if (!this.token) {
            window.location.href = '/login';
            return;
        }
        
        if (!this.user.is_admin) {
            alert('需要管理员权限');
            window.location.href = '/dashboard';
            return;
        }
    }
    
    bindEvents() {
        // 用户管理
        document.getElementById('addUserForm')?.addEventListener('submit', (e) => {
            e.preventDefault();
            this.addUser();
        });
        
        document.getElementById('editUserForm')?.addEventListener('submit', (e) => {
            e.preventDefault();
            this.updateUser();
        });
        
        // 系统设置
        document.getElementById('emailSettingsForm')?.addEventListener('submit', (e) => {
            e.preventDefault();
            this.saveEmailSettings();
        });
        
        document.getElementById('securitySettingsForm')?.addEventListener('submit', (e) => {
            e.preventDefault();
            this.saveSecuritySettings();
        });
        
        // 系统维护按钮
        document.getElementById('backupBtn')?.addEventListener('click', () => {
            this.backupDatabase();
        });
        
        document.getElementById('optimizeBtn')?.addEventListener('click', () => {
            this.optimizeDatabase();
        });
        
        document.getElementById('clearCacheBtn')?.addEventListener('click', () => {
            this.clearCache();
        });
        
        // 日志管理
        document.getElementById('refreshLogsBtn')?.addEventListener('click', () => {
            this.loadLogs();
        });
        
        document.getElementById('clearLogsBtn')?.addEventListener('click', () => {
            this.clearLogs();
        });
        
        document.getElementById('logLevel')?.addEventListener('change', () => {
            this.loadLogs();
        });
        
        // 系统通知
        document.getElementById('sendNotificationForm')?.addEventListener('submit', (e) => {
            e.preventDefault();
            this.sendSystemNotification();
        });
        
        // 邮件搜索
        document.getElementById('searchEmailsBtn')?.addEventListener('click', () => {
            this.searchEmails();
        });
        
        document.getElementById('emailSearch')?.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                this.searchEmails();
            }
        });
        
        // 刷新按钮
        document.getElementById('refreshEmailsBtn')?.addEventListener('click', () => {
            this.loadEmails();
        });
        
        // 通知目标切换
        document.querySelectorAll('input[name="notificationTarget"]').forEach(radio => {
            radio.addEventListener('change', (e) => {
                const userSelection = document.getElementById('userSelection');
                if (e.target.value === 'selected') {
                    userSelection.style.display = 'block';
                    this.loadUsersForNotification();
                } else {
                    userSelection.style.display = 'none';
                }
            });
        });
    }
    
    setupTabSwitching() {
        // 从URL获取标签页
        const hash = window.location.hash;
        if (hash) {
            const tab = document.querySelector(`[href="${hash}"]`);
            if (tab) {
                const bsTab = new bootstrap.Tab(tab);
                bsTab.show();
            }
        }
        
        // 标签页切换时更新URL
        document.querySelectorAll('[data-bs-toggle="tab"]').forEach(tab => {
            tab.addEventListener('shown.bs.tab', (e) => {
                const target = e.target.getAttribute('href');
                window.location.hash = target;
                
                // 加载对应标签页的数据
                switch(target) {
                    case '#users':
                        this.loadUsers();
                        break;
                    case '#emails':
                        this.loadEmails();
                        break;
                    case '#logs':
                        this.loadLogs();
                        break;
                    case '#notifications':
                        this.loadNotifications();
                        break;
                }
            });
        });
    }
    
    setupDataTables() {
        // 初始化用户表格
        if (document.getElementById('usersTable')) {
            this.usersTable = $('#usersTable').DataTable({
                pageLength: 10,
                lengthMenu: [10, 25, 50, 100],
                order: [[0, 'desc']],
                language: {
                    url: '//cdn.datatables.net/plug-ins/1.11.5/i18n/zh-CN.json'
                },
                columns: [
                    { data: 'id' },
                    { data: 'username' },
                    { data: 'email' },
                    { 
                        data: 'is_admin',
                        render: (data) => data ? 
                            '<span class="badge bg-success">是</span>' : 
                            '<span class="badge bg-secondary">否</span>'
                    },
                    { 
                        data: 'is_active',
                        render: (data) => data ? 
                            '<span class="badge bg-success">活跃</span>' : 
                            '<span class="badge bg-danger">禁用</span>'
                    },
                    { 
                        data: 'storage',
                        render: (data) => `
                            <div class="progress" style="height: 6px;">
                                <div class="progress-bar" 
                                     style="width: ${data.percent}%">
                                </div>
                            </div>
                            <small>${data.used.toFixed(1)} / ${data.max.toFixed(1)} GB</small>
                        `
                    },
                    { data: 'created_at' },
                    {
                        data: 'id',
                        orderable: false,
                        render: (data, type, row) => `
                            <button class="btn btn-sm btn-warning me-1" onclick="adminPanel.editUser(${data})">
                                <i class="fas fa-edit"></i>
                            </button>
                            <button class="btn btn-sm btn-danger" onclick="adminPanel.deleteUser(${data})">
                                <i class="fas fa-trash"></i>
                            </button>
                        `
                    }
                ]
            });
        }
        
        // 初始化邮件表格
        if (document.getElementById('emailsTable')) {
            this.emailsTable = $('#emailsTable').DataTable({
                pageLength: 10,
                lengthMenu: [10, 25, 50, 100],
                order: [[0, 'desc']],
                language: {
                    url: '//cdn.datatables.net/plug-ins/1.11.5/i18n/zh-CN.json'
                },
                columns: [
                    { data: 'id' },
                    { 
                        data: 'sender',
                        render: (data) => `
                            <div>
                                <strong>${this.escapeHtml(data.name)}</strong><br>
                                <small class="text-muted">${this.escapeHtml(data.email)}</small>
                            </div>
                        `
                    },
                    { 
                        data: 'recipient',
                        render: (data) => `
                            <div>
                                <strong>${this.escapeHtml(data.name)}</strong><br>
                                <small class="text-muted">${this.escapeHtml(data.email)}</small>
                            </div>
                        `
                    },
                    { data: 'subject' },
                    { 
                        data: 'status',
                        render: (data) => {
                            let badges = [];
                            if (data.is_read) badges.push('<span class="badge bg-success">已读</span>');
                            if (data.is_starred) badges.push('<span class="badge bg-warning">星标</span>');
                            if (status.is_deleted) badges.push('<span class="badge bg-danger">已删除</span>');
                            if (status.has_attachment) badges.push('<span class="badge bg-info">有附件</span>');
                            return badges.join(' ');
                        }
                    },
                    { data: 'time' },
                    {
                        data: 'id',
                        orderable: false,
                        render: (data) => `
                            <button class="btn btn-sm btn-info me-1" onclick="adminPanel.viewEmail(${data})">
                                <i class="fas fa-eye"></i>
                            </button>
                            <button class="btn btn-sm btn-danger" onclick="adminPanel.deleteEmailAdmin(${data})">
                                <i class="fas fa-trash"></i>
                            </button>
                        `
                    }
                ]
            });
        }
    }
    
    async loadDashboardStats() {
        try {
            const response = await fetch('/api/admin/stats', {
                headers: {
                    'Authorization': `Bearer ${this.token}`
                }
            });
            
            const data = await response.json();
            if (data.success) {
                const stats = data.stats;
                
                // 更新统计卡片
                document.getElementById('totalUsers').textContent = stats.users.total;
                document.getElementById('totalEmails').textContent = stats.emails.total;
                document.getElementById('storageUsed').textContent = stats.storage.used.toFixed(1) + ' GB';
                document.getElementById('onlineUsers').textContent = stats.users.active;
                
                // 更新系统信息
                document.getElementById('systemVersion').textContent = '1.0.0';
                document.getElementById('uptime').textContent = '0天0小时';
                document.getElementById('domain').textContent = stats.system.domain;
                document.getElementById('port').textContent = stats.system.port;
                document.getElementById('ssl').textContent = stats.system.ssl_enabled ? '已启用' : '未启用';
                document.getElementById('websocket').textContent = stats.system.websocket ? '已启用' : '未启用';
            }
        } catch (error) {
            console.error('加载统计数据错误:', error);
        }
    }
    
    async loadUsers() {
        try {
            const response = await fetch('/api/admin/users', {
                headers: {
                    'Authorization': `Bearer ${this.token}`
                }
            });
            
            const data = await response.json();
            if (data.success && this.usersTable) {
                this.usersTable.clear();
                this.usersTable.rows.add(data.users);
                this.usersTable.draw();
            }
        } catch (error) {
            console.error('加载用户列表错误:', error);
            this.showAlert('加载用户列表失败', 'danger');
        }
    }
    
    async loadEmails() {
        try {
            const search = document.getElementById('emailSearch')?.value || '';
            const url = search ? 
                `/api/admin/emails?search=${encodeURIComponent(search)}` :
                '/api/admin/emails';
            
            const response = await fetch(url, {
                headers: {
                    'Authorization': `Bearer ${this.token}`
                }
            });
            
            const data = await response.json();
            if (data.success && this.emailsTable) {
                // 转换数据格式
                const emails = data.emails.map(email => ({
                    id: email.id,
                    sender: {
                        name: email.sender_name,
                        email: email.sender_email
                    },
                    recipient: {
                        name: email.recipient_name,
                        email: email.recipient_email
                    },
                    subject: email.subject,
                    status: {
                        is_read: email.is_read,
                        is_starred: email.is_starred,
                        is_deleted: email.is_deleted,
                        has_attachment: email.has_attachment
                    },
                    time: email.time_ago
                }));
                
                this.emailsTable.clear();
                this.emailsTable.rows.add(emails);
                this.emailsTable.draw();
            }
        } catch (error) {
            console.error('加载邮件列表错误:', error);
            this.showAlert('加载邮件列表失败', 'danger');
        }
    }
    
    async addUser() {
        const form = document.getElementById('addUserForm');
        const formData = new FormData(form);
        
        const userData = {
            username: formData.get('username'),
            email: formData.get('email'),
            password: formData.get('password'),
            is_admin: formData.get('is_admin') === 'on',
            is_active: formData.get('is_active') === 'on',
            custom_domain: formData.get('custom_domain'),
            max_storage: parseInt(formData.get('max_storage')) * 1024 * 1024 * 1024 // 转换为字节
        };
        
        try {
            const response = await fetch('/api/admin/users', {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${this.token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(userData)
            });
            
            const data = await response.json();
            if (data.success) {
                this.showAlert('用户添加成功', 'success');
                this.loadUsers();
                
                // 关闭模态框
                const modal = bootstrap.Modal.getInstance(document.getElementById('addUserModal'));
                modal.hide();
                form.reset();
            } else {
                this.showAlert('添加失败: ' + data.message, 'danger');
            }
        } catch (error) {
            console.error('添加用户错误:', error);
            this.showAlert('网络错误，请稍后重试', 'danger');
        }
    }
    
    async editUser(userId) {
        try {
            // 获取用户信息
            const response = await fetch(`/api/admin/users/${userId}`, {
                headers: {
                    'Authorization': `Bearer ${this.token}`
                }
            });
            
            const data = await response.json();
            if (data.success) {
                const user = data.user;
                
                // 填充表单
                document.getElementById('editUserId').value = user.id;
                document.getElementById('editUsername').value = user.username;
                document.getElementById('editEmail').value = user.email;
                document.getElementById('editIsAdmin').checked = user.is_admin;
                document.getElementById('editIsActive').checked = user.is_active;
                document.getElementById('editCustomDomain').value = user.custom_domain || '';
                document.getElementById('editMaxStorage').value = Math.floor(user.max_storage / (1024 * 1024 * 1024));
                
                // 显示模态框
                const modal = new bootstrap.Modal(document.getElementById('editUserModal'));
                modal.show();
            }
        } catch (error) {
            console.error('获取用户信息错误:', error);
            this.showAlert('获取用户信息失败', 'danger');
        }
    }
    
    async updateUser() {
        const form = document.getElementById('editUserForm');
        const formData = new FormData(form);
        const userId = formData.get('id');
        
        const updateData = {
            username: formData.get('username'),
            email: formData.get('email'),
            is_admin: formData.get('is_admin') === 'on',
            is_active: formData.get('is_active') === 'on',
            custom_domain: formData.get('custom_domain'),
            max_storage: parseInt(formData.get('max_storage')) * 1024 * 1024 * 1024
        };
        
        // 如果有密码，添加到更新数据
        const password = formData.get('password');
        if (password) {
            updateData.password = password;
        }
        
        try {
            const response = await fetch(`/api/admin/users/${userId}`, {
                method: 'PUT',
                headers: {
                    'Authorization': `Bearer ${this.token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(updateData)
            });
            
            const data = await response.json();
            if (data.success) {
                this.showAlert('用户信息更新成功', 'success');
                this.loadUsers();
                
                // 关闭模态框
                const modal = bootstrap.Modal.getInstance(document.getElementById('editUserModal'));
                modal.hide();
            } else {
                this.showAlert('更新失败: ' + data.message, 'danger');
            }
        } catch (error) {
            console.error('更新用户错误:', error);
            this.showAlert('网络错误，请稍后重试', 'danger');
        }
    }
    
    async deleteUser(userId) {
        if (!confirm('确定要删除这个用户吗？此操作将永久删除用户的所有数据，包括邮件和附件。')) {
            return;
        }
        
        try {
            const response = await fetch(`/api/admin/users/${userId}`, {
                method: 'DELETE',
                headers: {
                    'Authorization': `Bearer ${this.token}`
                }
            });
            
            const data = await response.json();
            if (data.success) {
                this.showAlert('用户删除成功', 'success');
                this.loadUsers();
            } else {
                this.showAlert('删除失败: ' + data.message, 'danger');
            }
        } catch (error) {
            console.error('删除用户错误:', error);
            this.showAlert('网络错误，请稍后重试', 'danger');
        }
    }
    
    async viewEmail(emailId) {
        window.open(`/email/${emailId}`, '_blank');
    }
    
    async deleteEmailAdmin(emailId) {
        if (!confirm('确定要删除这封邮件吗？此操作将永久删除邮件。')) {
            return;
        }
        
        try {
            const response = await fetch(`/api/emails/${emailId}?permanent=true`, {
                method: 'DELETE',
                headers: {
                    'Authorization': `Bearer ${this.token}`
                }
            });
            
            const data = await response.json();
            if (data.success) {
                this.showAlert('邮件删除成功', 'success');
                this.loadEmails();
            } else {
                this.showAlert('删除失败: ' + data.message, 'danger');
            }
        } catch (error) {
            console.error('删除邮件错误:', error);
            this.showAlert('网络错误，请稍后重试', 'danger');
        }
    }
    
    async searchEmails() {
        this.loadEmails();
    }
    
    async loadLogs() {
        const level = document.getElementById('logLevel').value;
        const date = document.getElementById('logDate').value;
        
        try {
            const response = await fetch('/api/admin/logs', {
                headers: {
                    'Authorization': `Bearer ${this.token}`
                }
            });
            
            const data = await response.json();
            if (data.success) {
                const logsContent = document.getElementById('logsContent');
                logsContent.innerHTML = data.logs.map(log => `
                    <div class="alert alert-info mb-2">
                        <small>${this.escapeHtml(log)}</small>
                    </div>
                `).join('');
                
                // 滚动到底部
                logsContent.parentNode.scrollTop = logsContent.parentNode.scrollHeight;
            }
        } catch (error) {
            console.error('加载日志错误:', error);
            this.showAlert('加载日志失败', 'danger');
        }
    }
    
    async clearLogs() {
        if (!confirm('确定要清空所有系统日志吗？此操作不可恢复。')) {
            return;
        }
        
        try {
            const response = await fetch('/api/admin/logs/clear', {
                method: 'DELETE',
                headers: {
                    'Authorization': `Bearer ${this.token}`
                }
            });
            
            const data = await response.json();
            if (data.success) {
                this.showAlert('日志已清空', 'success');
                this.loadLogs();
            } else {
                this.showAlert('清空失败: ' + data.message, 'danger');
            }
        } catch (error) {
            console.error('清空日志错误:', error);
            this.showAlert('网络错误，请稍后重试', 'danger');
        }
    }
    
    async loadUsersForNotification() {
        try {
            const response = await fetch('/api/admin/users?limit=100', {
                headers: {
                    'Authorization': `Bearer ${this.token}`
                }
            });
            
            const data = await response.json();
            if (data.success) {
                const select = document.getElementById('selectedUsers');
                select.innerHTML = data.users.map(user => `
                    <option value="${user.id}">
                        ${this.escapeHtml(user.username)} (${this.escapeHtml(user.email)})
                    </option>
                `).join('');
            }
        } catch (error) {
            console.error('加载用户列表错误:', error);
        }
    }
    
    async sendSystemNotification() {
        const form = document.getElementById('sendNotificationForm');
        const formData = new FormData(form);
        
        const target = document.querySelector('input[name="notificationTarget"]:checked').value;
        const notificationData = {
            title: formData.get('notificationTitle'),
            message: formData.get('notificationMessage'),
            type: formData.get('notificationType')
        };
        
        if (target === 'all') {
            notificationData.to_all = true;
        } else {
            const selectedUsers = Array.from(document.getElementById('selectedUsers').selectedOptions)
                .map(option => parseInt(option.value));
            notificationData.user_ids = selectedUsers;
        }
        
        try {
            const response = await fetch('/api/admin/notifications', {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${this.token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(notificationData)
            });
            
            const data = await response.json();
            if (data.success) {
                this.showAlert('系统通知发送成功', 'success');
                form.reset();
            } else {
                this.showAlert('发送失败: ' + data.message, 'danger');
            }
        } catch (error) {
            console.error('发送通知错误:', error);
            this.showAlert('网络错误，请稍后重试', 'danger');
        }
    }
    
    async loadNotifications() {
        try {
            const response = await fetch('/api/admin/notifications', {
                headers: {
                    'Authorization': `Bearer ${this.token}`
                }
            });
            
            const data = await response.json();
            if (data.success) {
                // 更新通知列表
                const notificationsList = document.querySelector('#notifications .list-group');
                if (notificationsList) {
                    notificationsList.innerHTML = data.notifications.map(notification => `
                        <div class="list-group-item">
                            <div class="d-flex w-100 justify-content-between">
                                <h6 class="mb-1">${this.escapeHtml(notification.title)}</h6>
                                <small>${notification.time}</small>
                            </div>
                            <p class="mb-1">${this.escapeHtml(notification.message)}</p>
                            <small>发送给: ${notification.target}</small>
                        </div>
                    `).join('');
                }
            }
        } catch (error) {
            console.error('加载通知列表错误:', error);
        }
    }
    
    async saveEmailSettings() {
        const form = document.getElementById('emailSettingsForm');
        const formData = new FormData(form);
        
        const settings = {
            max_email_size: parseInt(formData.get('maxEmailSize')) * 1024 * 1024,
            default_storage: parseInt(formData.get('defaultStorage')) * 1024 * 1024 * 1024,
            daily_limit: parseInt(formData.get('dailyLimit'))
        };
        
        try {
            const response = await fetch('/api/admin/settings/email', {
                method: 'PUT',
                headers: {
                    'Authorization': `Bearer ${this.token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(settings)
            });
            
            const data = await response.json();
            if (data.success) {
                this.showAlert('邮件设置保存成功', 'success');
            } else {
                this.showAlert('保存失败: ' + data.message, 'danger');
            }
        } catch (error) {
            console.error('保存邮件设置错误:', error);
            this.showAlert('网络错误，请稍后重试', 'danger');
        }
    }
    
    async saveSecuritySettings() {
        const form = document.getElementById('securitySettingsForm');
        const formData = new FormData(form);
        
        const settings = {
            token_expiry: parseInt(formData.get('tokenExpiry')),
            rate_limit: parseInt(formData.get('rateLimit')),
            require_2fa: formData.get('require2FA') === 'on'
        };
        
        try {
            const response = await fetch('/api/admin/settings/security', {
                method: 'PUT',
                headers: {
                    'Authorization': `Bearer ${this.token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(settings)
            });
            
            const data = await response.json();
            if (data.success) {
                this.showAlert('安全设置保存成功', 'success');
            } else {
                this.showAlert('保存失败: ' + data.message, 'danger');
            }
        } catch (error) {
            console.error('保存安全设置错误:', error);
            this.showAlert('网络错误，请稍后重试', 'danger');
        }
    }
    
    async backupDatabase() {
        if (!confirm('确定要备份数据库吗？')) {
            return;
        }
        
        try {
            const response = await fetch('/api/admin/database/backup', {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${this.token}`
                }
            });
            
            const data = await response.json();
            if (data.success) {
                this.showAlert('数据库备份成功: ' + data.filename, 'success');
            } else {
                this.showAlert('备份失败: ' + data.message, 'danger');
            }
        } catch (error) {
            console.error('备份数据库错误:', error);
            this.showAlert('网络错误，请稍后重试', 'danger');
        }
    }
    
    async optimizeDatabase() {
        if (!confirm('确定要优化数据库吗？这可能需要一些时间。')) {
            return;
        }
        
        try {
            const response = await fetch('/api/admin/database/optimize', {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${this.token}`
                }
            });
            
            const data = await response.json();
            if (data.success) {
                this.showAlert('数据库优化完成', 'success');
            } else {
                this.showAlert('优化失败: ' + data.message, 'danger');
            }
        } catch (error) {
            console.error('优化数据库错误:', error);
            this.showAlert('网络错误，请稍后重试', 'danger');
        }
    }
    
    async clearCache() {
        if (!confirm('确定要清理所有缓存吗？')) {
            return;
        }
        
        try {
            const response = await fetch('/api/admin/cache/clear', {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${this.token}`
                }
            });
            
            const data = await response.json();
            if (data.success) {
                this.showAlert('缓存清理完成', 'success');
            } else {
                this.showAlert('清理失败: ' + data.message, 'danger');
            }
        } catch (error) {
            console.error('清理缓存错误:', error);
            this.showAlert('网络错误，请稍后重试', 'danger');
        }
    }
    
    showAlert(message, type = 'info') {
        const alertDiv = document.createElement('div');
        alertDiv.className = `alert alert-${type} alert-dismissible fade show position-fixed`;
        alertDiv.style.cssText = `
            top: 20px;
            right: 20px;
            z-index: 9999;
            min-width: 300px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        `;
        
        alertDiv.innerHTML = `
            <i class="fas fa-${type === 'success' ? 'check-circle' : type === 'danger' ? 'exclamation-circle' : 'info-circle'} me-2"></i>
            ${this.escapeHtml(message)}
            <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
        `;
        
        document.body.appendChild(alertDiv);
        
        // 5秒后自动移除
        setTimeout(() => {
            if (alertDiv.parentNode) {
                alertDiv.parentNode.removeChild(alertDiv);
            }
        }, 5000);
    }
    
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// 页面加载完成后初始化管理员面板
document.addEventListener('DOMContentLoaded', () => {
    window.adminPanel = new AdminPanel();
});