
see https://github.com/operator-framework/operator-sdk

needs prom 
https://github.com/coreos/prometheus-operator

https://banzaicloud.com/blog/operator-sdk/



	kubectl create -f deploy/service_account.yaml
	kubectl create -f deploy/role.yaml
	kubectl create -f deploy/role_binding.yaml

	kubectl create -f deploy/crds/app.knotfree.io_appservices_crd.yaml

	kk apply -f  deploy/promethius_op.yaml 
	kubectl create -f deploy/crds/app.knotfree.io_v1alpha1_appservice_cr.yaml
	

.Watch(

