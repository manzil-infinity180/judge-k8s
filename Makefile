dev:
	# 1. Delete everything
	# k3d cluster delete dev 2>/dev/null || true
	# docker stop myreg && docker rm myreg

	# 2. Create registry with k3d (so it gets the right name)
	k3d registry create myregistry --port 34751

	# 3. Verify registry name
	docker ps | grep registry
	# Should show: k3d-myregistry

	# 4. Create registries.yaml
	# cat > /Users/rahulxf/JourneyToXYZ/k8s-witness-demo/judge-k8s/registries.yaml << 'EOF'
	# mirrors:
	#   "localhost:34751":
	#     endpoint:
	#       - http://k3d-myregistry:5000
	# EOF

	# k3d cluster delete dev

	# 5. Create cluster
	k3d cluster create dev \
	  --registry-use k3d-myregistry:5000 \
	  --registry-config "/Users/rahulxf/JourneyToXYZ/k8s-witness-demo/judge-k8s/registries.yaml"
	  # --volume ~/k8s:/k8s@all

	# 6. Verify configuration
	docker exec k3d-dev-server-0 cat /etc/rancher/k3s/registries.yaml
