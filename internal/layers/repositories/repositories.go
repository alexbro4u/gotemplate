package repositories

type Repositories struct {
	User         UserRepository
	UserGroup    UserGroupRepository
	RequestCache RequestCacheRepository
}
