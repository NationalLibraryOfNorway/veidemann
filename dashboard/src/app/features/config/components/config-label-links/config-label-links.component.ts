import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {MatListModule} from '@angular/material/list';

import {ResolvedLabelLink} from '../../func';

@Component({
  selector: 'app-config-label-links',
  templateUrl: './config-label-links.component.html',
  styleUrls: ['./config-label-links.component.scss'],
  imports: [MatListModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
})
export class ConfigLabelLinksComponent {
  @Input() links: readonly ResolvedLabelLink[] = [];

  linkAriaLabel(text: string): string {
    return $localize`:@@configLabelLinkOpenAriaLabel:Open ${text}:LINK_TITLE: in a new tab`;
  }
}
