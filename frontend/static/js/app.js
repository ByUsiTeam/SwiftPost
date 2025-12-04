// SwiftPost 前端应用主文件

class SwiftPostApp {
    constructor() {
        this.token = localStorage.getItem('token');
        this.user = JSON.parse(localStorage.getItem('user') || '{}');
        this.currentFolder = 'inbox';
        this.currentPage = 1;
        this.emails = [];
        this.totalEmails = 0;
        this.websocket = null;
        
        this.init();
    }
    
    init() {
        this.bindEvents();
        this.checkAuth();
        this.updateUI();
        
        if (this.token && this.user.id) {
            this.loadEmails();
            this.connectWebSocket();
        }
    }
    
    bindEvents() {
        // 导航点击
        document.addEventListener('click', (e) => {
            if (e.target.closest('.folder-nav')) {
                e.preventDefault();
                const folder = e.target.dataset.folder;
                if (folder) {
                    this.switchFolder(folder);
                }
            }
            
            if (e.target.closest('.compose-btn')) {
                e.preventDefault();
                this.showComposeModal();
            }
            
            if (e.target.closest('.refresh-btn')) {
                e.preventDefault();
                this.loadEmails();
            }
            
            if (e.target.closest('.logout-btn')) {
                e.preventDefault();
                this.logout();
            }
            
            // 邮件操作
            if (e.target.closest('.email-checkbox')) {
                this.updateSelection();
            }
            
            if (e.target.closest('.select-all-checkbox')) {
                this.selectAllEmails(e.target.checked);
            }
            
            if (e.target.closest('.view-email-btn')) {
                e.preventDefault();
                const emailId = e.target.dataset.emailId || e.target.closest('.view-email-btn').dataset.emailId;
                this.viewEmail(emailId);
            }
            
            if (e.target.closest('.star-email-btn')) {
                e.preventDefault();
                const emailId = e.target.dataset.emailId;
                this.toggleStar(emailId);
            }
            
            if (e.target.closest('.delete-email-btn')) {
                e.preventDefault();
                const emailId = e.target.dataset.emailId;
                this.deleteEmail(emailId);
            }
        });
        
        // 搜索表单
        const searchForm = document.getElementById('searchForm');
        if (searchForm) {
            searchForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.searchEmails();
            });
        }
        
        // 发送邮件表单
        const sendEmailForm = document.getElementById('sendEmailForm');
        if (sendEmailForm) {
            sendEmailForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.sendEmail();
            });
        }
        
        // 回复邮件
        const replyBtn = document.getElementById('replyBtn');
        if (replyBtn) {
            replyBtn.addEventListener('click', () => {
                this.replyToEmail();
            });
        }
        
        // 转发邮件
        const forwardBtn = document.getElementById('forwardBtn');
        if (forwardBtn) {
            forwardBtn.addEventListener('click', () => {
                this.forwardEmail();
            });
        }
        
        // 分页
        document.addEventListener('click', (e) => {
            if (e.target.closest('.page-link')) {
                e.preventDefault();
                const page = parseInt(e.target.dataset.page);
                if (page) {
                    this.goToPage(page);
                }
            }
        });
    }
    
    checkAuth() {
        if (!this.token) {
            // 检查URL路径
            const path = window.location.pathname;
            const publicPaths = ['/', '/login', '/register', '/blocked'];
            
            if (!publicPaths.includes(path)) {
                window.location.href = '/login';
            }
        }
    }
    
    updateUI() {
        // 更新用户信息
        const userElements = document.querySelectorAll('.user-info');
        userElements.forEach(el => {
            if (el.dataset.field === 'username' && this.user.username) {
                el.textContent = this.user.username;
            }
            if (el.dataset.field === 'email' && this.user.email) {
                el.textContent = this.user.email;
            }
        });
        
        // 更新未读计数
        this.updateUnreadCount();
        
        // 更新存储使用情况
        this.updateStorageUsage();
    }
    
    async loadEmails() {
        try {
            const response = await fetch(`/api/emails?folder=${this.currentFolder}&page=${this.currentPage}&limit=20`, {
                headers: {
                    'Authorization': `Bearer ${this.token}`
                }
            });
            
            if (!response.ok) {
                if (response.status === 401) {
                    this.logout();
                    return;
                }
                throw new Error('加载邮件失败');
            }
            
            const data = await response.json();
            this.emails = data.emails || [];
            this.totalEmails = data.pagination?.total || 0;
            
            this.renderEmailList();
            this.renderPagination();
            
        } catch (error) {
            console.error('加载邮件错误:', error);
            this.showAlert('加载邮件失败，请重试', 'danger');
        }
    }
    
    renderEmailList() {
        const emailList = document.getElementById('emailList');
        if (!emailList) return;
        
        if (this.emails.length === 0) {
            emailList.innerHTML = `
                <tr>
                    <td colspan="6" class="text-center py-5">
                        <i class="fas fa-inbox fa-3x text-muted mb-3"></i>
                        <p class="text-muted">没有邮件</p>
                    </td>
                </tr>
            `;
            return;
        }
        
        emailList.innerHTML = this.emails.map(email => `
            <tr class="email-item ${email.is_read ? '' : 'unread'}" data-email-id="${email.id}">
                <td style="width: 40px;">
                    <input type="checkbox" class="email-checkbox form-check-input" value="${email.id}">
                </td>
                <td style="width: 40px;">
                    <button class="btn btn-link btn-sm p-0 star-email-btn" data-email-id="${email.id}" title="${email.is_starred ? '取消星标' : '标记星标'}">
                        <i class="fas fa-star ${email.is_starred ? 'text-warning' : 'text-muted'}"></i>
                    </button>
                </td>
                <td class="email-sender" style="width: 200px;">
                    <strong>${this.escapeHtml(email.sender_name)}</strong>
                </td>
                <td class="email-subject" style="min-width: 200px;">
                    <a href="/email/${email.id}" class="text-decoration-none view-email-btn" data-email-id="${email.id}">
                        ${this.escapeHtml(email.subject)}
                        ${email.has_attachment ? '<i class="fas fa-paperclip ms-2 text-muted"></i>' : ''}
                    </a>
                </td>
                <td class="email-preview text-muted" style="min-width: 300px;">
                    ${this.escapeHtml(email.body_preview || '')}
                </td>
                <td class="email-time text-muted text-end" style="width: 150px;">
                    ${email.time_ago}
                </td>
            </tr>
        `).join('');
    }
    
    renderPagination() {
        const pagination = document.getElementById('pagination');
        if (!pagination) return;
        
        const totalPages = Math.ceil(this.totalEmails / 20);
        if (totalPages <= 1) {
            pagination.innerHTML = '';
            return;
        }
        
        let paginationHTML = '';
        
        // 上一页按钮
        if (this.currentPage > 1) {
            paginationHTML += `
                <li class="page-item">
                    <a class="page-link" href="#" data-page="${this.currentPage - 1}">
                        <i class="fas fa-chevron-left"></i>
                    </a>
                </li>
            `;
        }
        
        // 页码按钮
        const maxVisiblePages = 5;
        let startPage = Math.max(1, this.currentPage - Math.floor(maxVisiblePages / 2));
        let endPage = Math.min(totalPages, startPage + maxVisiblePages - 1);
        
        if (endPage - startPage + 1 < maxVisiblePages) {
            startPage = Math.max(1, endPage - maxVisiblePages + 1);
        }
        
        for (let i = startPage; i <= endPage; i++) {
            paginationHTML += `
                <li class="page-item ${i === this.currentPage ? 'active' : ''}">
                    <a class="page-link" href="#" data-page="${i}">${i}</a>
                </li>
            `;
        }
        
        // 下一页按钮
        if (this.currentPage < totalPages) {
            paginationHTML += `
                <li class="page-item">
                    <a class="page-link" href="#" data-page="${this.currentPage + 1}">
                        <i class="fas fa-chevron-right"></i>
                    </a>
                </li>
            `;
        }
        
        pagination.innerHTML = paginationHTML;
    }
    
    async viewEmail(emailId) {
        try {
            const response = await fetch(`/api/emails/${emailId}`, {
                headers: {
                    'Authorization': `Bearer ${this.token}`
                }
            });
            
            if (!response.ok) {
                throw new Error('加载邮件失败');
            }
            
            const data = await response.json();
            if (!data.success) {
                throw new Error(data.message || '加载邮件失败');
            }
            
            this.showEmailModal(data.email);
            
        } catch (error) {
            console.error('查看邮件错误:', error);
            this.showAlert('加载邮件失败', 'danger');
        }
    }
    
    showEmailModal(email) {
        // 创建模态框
        const modalHTML = `
            <div class="modal fade" id="emailModal" tabindex="-1" aria-hidden="true">
                <div class="modal-dialog modal-lg">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title">${this.escapeHtml(email.subject)}</h5>
                            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                        </div>
                        <div class="modal-body">
                            <div class="email-header mb-4">
                                <div class="d-flex justify-content-between align-items-start mb-2">
                                    <div>
                                        <strong>发件人:</strong> ${this.escapeHtml(email.sender_name)} &lt;${this.escapeHtml(email.sender_email)}&gt;
                                    </div>
                                    <div class="text-muted">
                                        ${email.time_ago}
                                    </div>
                                </div>
                                <div class="mb-2">
                                    <strong>收件人:</strong> ${this.escapeHtml(email.recipient_name)} &lt;${this.escapeHtml(email.recipient_email)}&gt;
                                </div>
                            </div>
                            
                            <div class="email-body mb-4">
                                ${email.body}
                            </div>
                            
                            ${email.attachments && email.attachments.length > 0 ? `
                            <div class="attachments mb-4">
                                <h6><i class="fas fa-paperclip me-2"></i>附件 (${email.attachments.length})</h6>
                                <div class="list-group">
                                    ${email.attachments.map(att => `
                                        <a href="/api/attachments/${att.uuid}/download" 
                                           class="list-group-item list-group-item-action d-flex justify-content-between align-items-center">
                                            <div>
                                                <i class="fas fa-file me-2"></i>
                                                ${this.escapeHtml(att.filename)}
                                            </div>
                                            <span class="badge bg-secondary rounded-pill">
                                                ${this.formatFileSize(att.file_size)}
                                            </span>
                                        </a>
                                    `).join('')}
                                </div>
                            </div>
                            ` : ''}
                        </div>
                        <div class="modal-footer">
                            <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">关闭</button>
                            <button type="button" class="btn btn-primary" id="replyBtn" data-email-id="${email.id}">
                                <i class="fas fa-reply me-1"></i>回复
                            </button>
                            <button type="button" class="btn btn-outline-primary" id="forwardBtn" data-email-id="${email.id}">
                                <i class="fas fa-share me-1"></i>转发
                            </button>
                            <button type="button" class="btn btn-danger" id="deleteEmailBtn" data-email-id="${email.id}">
                                <i class="fas fa-trash me-1"></i>删除
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        // 添加到页面
        const modalContainer = document.getElementById('modalContainer') || (() => {
            const div = document.createElement('div');
            div.id = 'modalContainer';
            document.body.appendChild(div);
            return div;
        })();
        
        modalContainer.innerHTML = modalHTML;
        
        // 显示模态框
        const modal = new bootstrap.Modal(document.getElementById('emailModal'));
        modal.show();
        
        // 绑定按钮事件
        document.getElementById('replyBtn').addEventListener('click', () => {
            modal.hide();
            this.replyToEmail(email);
        });
        
        document.getElementById('forwardBtn').addEventListener('click', () => {
            modal.hide();
            this.forwardEmail(email);
        });
        
        document.getElementById('deleteEmailBtn').addEventListener('click', () => {
            if (confirm('确定要删除这封邮件吗？')) {
                this.deleteEmail(email.id);
                modal.hide();
            }
        });
    }
    
    showComposeModal(email = null) {
        const isReply = email !== null;
        const modalHTML = `
            <div class="modal fade" id="composeModal" tabindex="-1" aria-hidden="true">
                <div class="modal-dialog modal-lg">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title">
                                ${isReply ? '回复邮件' : '撰写新邮件'}
                            </h5>
                            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                        </div>
                        <form id="sendEmailForm">
                            <div class="modal-body">
                                <div class="mb-3">
                                    <label for="recipient" class="form-label">收件人</label>
                                    <input type="email" class="form-control" id="recipient" 
                                           value="${isReply ? this.escapeHtml(email.sender_email) : ''}" 
                                           required multiple>
                                    <div class="form-text">多个邮箱用逗号分隔</div>
                                </div>
                                
                                <div class="mb-3">
                                    <label for="subject" class="form-label">主题</label>
                                    <input type="text" class="form-control" id="subject" 
                                           value="${isReply ? `Re: ${this.escapeHtml(email.subject)}` : ''}" 
                                           required>
                                </div>
                                
                                <div class="mb-3">
                                    <label for="emailBody" class="form-label">内容</label>
                                    <textarea class="form-control" id="emailBody" rows="10" required></textarea>
                                </div>
                                
                                <div class="mb-3">
                                    <label for="attachments" class="form-label">附件</label>
                                    <input type="file" class="form-control" id="attachments" multiple>
                                    <div class="form-text">最大单个文件25MB</div>
                                </div>
                                
                                ${isReply ? `
                                <div class="alert alert-info">
                                    <i class="fas fa-info-circle me-2"></i>
                                    正在回复来自 ${this.escapeHtml(email.sender_name)} 的邮件
                                </div>
                                ` : ''}
                            </div>
                            <div class="modal-footer">
                                <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">取消</button>
                                <button type="button" class="btn btn-success" id="saveDraftBtn">
                                    <i class="fas fa-save me-1"></i>保存草稿
                                </button>
                                <button type="submit" class="btn btn-primary">
                                    <i class="fas fa-paper-plane me-1"></i>发送邮件
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            </div>
        `;
        
        const modalContainer = document.getElementById('modalContainer') || (() => {
            const div = document.createElement('div');
            div.id = 'modalContainer';
            document.body.appendChild(div);
            return div;
        })();
        
        modalContainer.innerHTML = modalHTML;
        
        const modal = new bootstrap.Modal(document.getElementById('composeModal'));
        modal.show();
        
        // 如果是回复，预填内容
        if (isReply) {
            const originalDate = new Date(email.created_at).toLocaleString();
            const quotedText = `

----------------------------------------
在 ${originalDate}，${email.sender_name} 写道：

> ${email.body.replace(/\n/g, '\n> ')}
`;
            
            document.getElementById('emailBody').value = quotedText;
        }
        
        // 绑定表单提交
        const form = document.getElementById('sendEmailForm');
        form.addEventListener('submit', (e) => {
            e.preventDefault();
            this.sendEmail();
        });
        
        // 绑定保存草稿
        document.getElementById('saveDraftBtn').addEventListener('click', () => {
            this.saveAsDraft();
        });
    }
    
    async sendEmail() {
        const form = document.getElementById('sendEmailForm');
        if (!form) return;
        
        const formData = new FormData();
        formData.append('to', document.getElementById('recipient').value);
        formData.append('subject', document.getElementById('subject').value);
        formData.append('body', document.getElementById('emailBody').value);
        
        // 添加附件
        const attachments = document.getElementById('attachments');
        if (attachments && attachments.files.length > 0) {
            for (let i = 0; i < attachments.files.length; i++) {
                formData.append('attachments', attachments.files[i]);
            }
        }
        
        try {
            const response = await fetch('/api/emails/send', {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${this.token}`
                },
                body: formData
            });
            
            const data = await response.json();
            
            if (data.success) {
                this.showAlert('邮件发送成功！', 'success');
                
                // 关闭模态框
                const modal = bootstrap.Modal.getInstance(document.getElementById('composeModal'));
                if (modal) {
                    modal.hide();
                }
                
                // 刷新邮件列表
                this.loadEmails();
                
            } else {
                this.showAlert(data.message || '发送失败', 'danger');
            }
            
        } catch (error) {
            console.error('发送邮件错误:', error);
            this.showAlert('发送失败，请检查网络连接', 'danger');
        }
    }
    
    async toggleStar(emailId) {
        try {
            const response = await fetch(`/api/emails/${emailId}/star`, {
                method: 'PUT',
                headers: {
                    'Authorization': `Bearer ${this.token}`,
                    'Content-Type': 'application/json'
                }
            });
            
            const data = await response.json();
            
            if (data.success) {
                this.loadEmails(); // 刷新列表
            }
            
        } catch (error) {
            console.error('标记星标错误:', error);
        }
    }
    
    async deleteEmail(emailId) {
        if (!confirm('确定要删除这封邮件吗？')) {
            return;
        }
        
        try {
            const response = await fetch(`/api/emails/${emailId}`, {
                method: 'DELETE',
                headers: {
                    'Authorization': `Bearer ${this.token}`
                }
            });
            
            const data = await response.json();
            
            if (data.success) {
                this.showAlert('邮件已删除', 'success');
                this.loadEmails(); // 刷新列表
            } else {
                this.showAlert(data.message || '删除失败', 'danger');
            }
            
        } catch (error) {
            console.error('删除邮件错误:', error);
            this.showAlert('删除失败', 'danger');
        }
    }
    
    switchFolder(folder) {
        this.currentFolder = folder;
        this.currentPage = 1;
        this.loadEmails();
        
        // 更新活动状态
        document.querySelectorAll('.folder-nav').forEach(nav => {
            nav.classList.remove('active');
        });
        document.querySelector(`[data-folder="${folder}"]`).classList.add('active');
        
        // 更新标题
        const folderNames = {
            'inbox': '收件箱',
            'sent': '已发送',
            'drafts': '草稿箱',
            'starred': '星标邮件',
            'trash': '回收站'
        };
        
        document.title = `${folderNames[folder] || folder} - SwiftPost`;
    }
    
    goToPage(page) {
        this.currentPage = page;
        this.loadEmails();
    }
    
    searchEmails() {
        const searchInput = document.getElementById('searchInput');
        if (!searchInput) return;
        
        const query = searchInput.value.trim();
        if (!query) return;
        
        // 这里实现搜索功能
        this.showAlert('搜索功能开发中', 'info');
    }
    
    updateUnreadCount() {
        const unreadCount = this.emails.filter(email => !email.is_read).length;
        const unreadBadge = document.getElementById('unreadBadge');
        if (unreadBadge) {
            unreadBadge.textContent = unreadCount;
            unreadBadge.style.display = unreadCount > 0 ? 'inline-block' : 'none';
        }
    }
    
    updateStorageUsage() {
        // 获取存储使用情况
        fetch('/api/user/stats', {
            headers: {
                'Authorization': `Bearer ${this.token}`
            }
        })
        .then(response => response.json())
        .then(data => {
            if (data.success && data.user?.storage) {
                const storage = data.user.storage;
                const progressBar = document.getElementById('storageProgress');
                const storageText = document.getElementById('storageText');
                
                if (progressBar) {
                    progressBar.style.width = `${storage.percent}%`;
                    progressBar.setAttribute('aria-valuenow', storage.percent);
                    
                    if (storage.percent > 90) {
                        progressBar.classList.add('bg-danger');
                    } else if (storage.percent > 70) {
                        progressBar.classList.add('bg-warning');
                    } else {
                        progressBar.classList.add('bg-success');
                    }
                }
                
                if (storageText) {
                    storageText.textContent = `${storage.used.toFixed(1)} MB / ${storage.max.toFixed(1)} MB`;
                }
            }
        })
        .catch(console.error);
    }
    
    connectWebSocket() {
        if (!this.user.id) return;
        
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws?user_id=${this.user.id}&token=${this.token}`;
        
        this.websocket = new WebSocket(wsUrl);
        
        this.websocket.onopen = () => {
            console.log('WebSocket连接已建立');
            this.showNotification('连接已建立', 'success');
        };
        
        this.websocket.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                this.handleWebSocketMessage(message);
            } catch (error) {
                console.error('解析WebSocket消息错误:', error);
            }
        };
        
        this.websocket.onclose = () => {
            console.log('WebSocket连接已关闭');
            // 5秒后重连
            setTimeout(() => {
                this.connectWebSocket();
            }, 5000);
        };
        
        this.websocket.onerror = (error) => {
            console.error('WebSocket错误:', error);
        };
    }
    
    handleWebSocketMessage(message) {
        switch (message.type) {
            case 'new_email':
                this.handleNewEmail(message.payload);
                break;
            case 'info':
                console.log('WebSocket信息:', message.payload);
                break;
            case 'presence':
                this.handlePresence(message.payload);
                break;
            default:
                console.log('未知WebSocket消息类型:', message.type);
        }
    }
    
    handleNewEmail(payload) {
        // 显示新邮件通知
        this.showNotification(`新邮件: ${payload.subject}`, 'info');
        
        // 播放提示音
        this.playNotificationSound();
        
        // 如果当前在收件箱，刷新列表
        if (this.currentFolder === 'inbox') {
            this.loadEmails();
        }
        
        // 更新未读计数
        this.updateUnreadCount();
    }
    
    handlePresence(payload) {
        // 更新在线状态显示
        const presenceElement = document.getElementById('userPresence');
        if (presenceElement) {
            const statusText = payload.status === 'online' ? '在线' : '离线';
            presenceElement.textContent = statusText;
            presenceElement.className = `badge bg-${payload.status === 'online' ? 'success' : 'secondary'}`;
        }
    }
    
    playNotificationSound() {
        try {
            const audio = new Audio('/static/sounds/notification.mp3');
            audio.volume = 0.3;
            audio.play().catch(() => {
                // 忽略播放错误
            });
        } catch (error) {
            console.error('播放提示音错误:', error);
        }
    }
    
    showNotification(message, type = 'info') {
        // 创建通知元素
        const notification = document.createElement('div');
        notification.className = `alert alert-${type} alert-dismissible fade show position-fixed`;
        notification.style.cssText = `
            top: 20px;
            right: 20px;
            z-index: 9999;
            min-width: 300px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        `;
        
        notification.innerHTML = `
            <i class="fas fa-${type === 'success' ? 'check-circle' : type === 'danger' ? 'exclamation-circle' : 'info-circle'} me-2"></i>
            ${this.escapeHtml(message)}
            <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
        `;
        
        document.body.appendChild(notification);
        
        // 5秒后自动移除
        setTimeout(() => {
            if (notification.parentNode) {
                notification.parentNode.removeChild(notification);
            }
        }, 5000);
    }
    
    showAlert(message, type = 'info') {
        const alertDiv = document.createElement('div');
        alertDiv.className = `alert alert-${type} alert-dismissible fade show`;
        alertDiv.innerHTML = `
            <i class="fas fa-${type === 'success' ? 'check-circle' : type === 'danger' ? 'exclamation-circle' : 'info-circle'} me-2"></i>
            ${this.escapeHtml(message)}
            <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
        `;
        
        const container = document.querySelector('.alert-container') || document.querySelector('.container');
        if (container) {
            container.insertBefore(alertDiv, container.firstChild);
        }
        
        // 5秒后自动移除
        setTimeout(() => {
            if (alertDiv.parentNode) {
                alertDiv.parentNode.removeChild(alertDiv);
            }
        }, 5000);
    }
    
    logout() {
        fetch('/api/logout', {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${this.token}`
            }
        })
        .finally(() => {
            // 清除本地存储
            localStorage.removeItem('token');
            localStorage.removeItem('user');
            
            // 关闭WebSocket连接
            if (this.websocket) {
                this.websocket.close();
            }
            
            // 跳转到登录页
            window.location.href = '/login';
        });
    }
    
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
    
    formatFileSize(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }
    
    updateSelection() {
        const selectedCount = document.querySelectorAll('.email-checkbox:checked').length;
        const selectAllCheckbox = document.querySelector('.select-all-checkbox');
        if (selectAllCheckbox) {
            selectAllCheckbox.indeterminate = selectedCount > 0 && selectedCount < this.emails.length;
            selectAllCheckbox.checked = selectedCount === this.emails.length;
        }
    }
    
    selectAllEmails(checked) {
        document.querySelectorAll('.email-checkbox').forEach(checkbox => {
            checkbox.checked = checked;
        });
    }
    
    replyToEmail(email) {
        this.showComposeModal(email);
    }
    
    forwardEmail(email) {
        const forwardedSubject = `Fw: ${email.subject}`;
        const forwardedBody = `

----------------------------------------
转发邮件:
发件人: ${email.sender_name} <${email.sender_email}>
日期: ${new Date(email.created_at).toLocaleString()}
收件人: ${email.recipient_name} <${email.recipient_email}>
主题: ${email.subject}

${email.body}
`;
        
        this.showComposeModal({
            subject: forwardedSubject,
            body: forwardedBody
        });
    }
    
    async saveAsDraft() {
        // 保存草稿功能
        this.showAlert('草稿保存功能开发中', 'info');
    }
}

// 页面加载完成后初始化应用
document.addEventListener('DOMContentLoaded', () => {
    window.swiftpost = new SwiftPostApp();
});

// 辅助函数
function formatDateTime(date) {
    if (!date) return '';
    
    const d = new Date(date);
    const now = new Date();
    const diff = now - d;
    
    // 如果是今天
    if (d.toDateString() === now.toDateString()) {
        return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    }
    
    // 如果是今年
    if (d.getFullYear() === now.getFullYear()) {
        return d.toLocaleDateString([], { month: 'short', day: 'numeric' });
    }
    
    // 其他情况
    return d.toLocaleDateString();
}

// 注册Service Worker（PWA支持）
if ('serviceWorker' in navigator) {
    window.addEventListener('load', () => {
        navigator.serviceWorker.register('/service-worker.js')
            .then(registration => {
                console.log('ServiceWorker注册成功:', registration.scope);
            })
            .catch(error => {
                console.log('ServiceWorker注册失败:', error);
            });
    });
}

// 离线检测
window.addEventListener('online', () => {
    window.swiftpost?.showNotification('网络连接已恢复', 'success');
});

window.addEventListener('offline', () => {
    window.swiftpost?.showNotification('网络连接已断开', 'warning');
});