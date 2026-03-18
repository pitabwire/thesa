/// Image renderer.
library;

import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';

import '../../../core/core.dart';
import '../../../widgets/shared/shared.dart';

/// Renders image component
class ImageRenderer extends StatelessWidget {
  const ImageRenderer({
    required this.component,
    super.key,
  });

  final ComponentDescriptor component;

  @override
  Widget build(BuildContext context) {
    final url = component.config['url'] as String?;
    final alt = component.config['alt'] as String? ??
        component.ui?.label ??
        'Image';
    final width = (component.config['width'] as num?)?.toDouble();
    final height = (component.config['height'] as num?)?.toDouble();
    final fit = _parseFit(component.config['fit'] as String?);

    if (url == null || url.isEmpty) {
      return Container(
        width: width ?? 200,
        height: height ?? 200,
        color: Colors.grey[300],
        child: const Center(
          child: Icon(Icons.image, size: 48),
        ),
      );
    }

    return CachedNetworkImage(
      imageUrl: url,
      width: width,
      height: height,
      fit: fit,
      placeholder: (context, url) => const AppLoadingIndicator(
        message: 'Loading image...',
      ),
      errorWidget: (context, url, error) => Container(
        width: width ?? 200,
        height: height ?? 200,
        color: Colors.grey[300],
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.broken_image, size: 48),
            Text(alt, style: const TextStyle(fontSize: 12)),
          ],
        ),
      ),
    );
  }

  BoxFit _parseFit(String? fit) {
    switch (fit?.toLowerCase()) {
      case 'contain':
        return BoxFit.contain;
      case 'cover':
        return BoxFit.cover;
      case 'fill':
        return BoxFit.fill;
      case 'fitwidth':
        return BoxFit.fitWidth;
      case 'fitheight':
        return BoxFit.fitHeight;
      case 'none':
        return BoxFit.none;
      case 'scaledown':
        return BoxFit.scaleDown;
      default:
        return BoxFit.cover;
    }
  }
}
